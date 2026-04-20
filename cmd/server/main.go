package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-ai-trading-engine/internal/app"
	"go-ai-trading-engine/internal/config"
	"go-ai-trading-engine/internal/domain"
	"go-ai-trading-engine/internal/middleware"
	"go-ai-trading-engine/internal/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	log := logger.New(cfg.AppEnv)
	container, err := app.Build(cfg, log)
	if err != nil {
		log.Error("build container failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := container.Close(); err != nil {
			log.Warn("container close error", "error", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ticks := make(chan domain.PriceTick, cfg.PriceChannelBuffer)
	container.WSClient.Start(ctx, ticks)
	container.MarketService.StartTickProcessors(ctx, ticks, 4)

	h := container.HTTPHandler.Router()
	rl := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	h = rl.Middleware(h)
	h = middleware.Recoverer(log, h)
	h = middleware.RequestLogger(log, h)

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      h,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		log.Info("http server starting", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", "error", err)
		if err := server.Close(); err != nil {
			log.Error("http force-close failed", "error", err)
		}
	}

	log.Info("service stopped", "time", time.Now().UTC())
}
