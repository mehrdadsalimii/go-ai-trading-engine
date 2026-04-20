package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"go-ai-trading-engine/internal/config"
	httpHandler "go-ai-trading-engine/internal/handler/http"
	"go-ai-trading-engine/internal/infrastructure/wsclient"
	"go-ai-trading-engine/internal/repository/postgres"
	redisrepo "go-ai-trading-engine/internal/repository/redis"
	"go-ai-trading-engine/internal/service/ai"
	"go-ai-trading-engine/internal/service/market"
)

type Container struct {
	DB            *sql.DB
	Redis         *redis.Client
	HTTPHandler   *httpHandler.Handler
	WSClient      *wsclient.BinanceClient
	MarketService *market.Service
}

func Build(cfg config.Config, logger *slog.Logger) (*Container, error) {
	db, err := postgres.Open(cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	priceRepo := redisrepo.NewPriceRepository(redisClient, int64(cfg.LatestPriceTTL.Seconds()))
	tradingRepo := postgres.NewTradingRepository(db)
	marketSvc := market.NewService(priceRepo, tradingRepo, logger, cfg.RecentHistoryMaxSize)
	openAIClient := ai.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	aiSvc := ai.NewService(openAIClient, tradingRepo, marketSvc, logger)
	h := httpHandler.NewHandler(marketSvc, aiSvc, logger)
	ws := wsclient.NewBinanceClient(cfg.BinanceWSURL, logger)

	return &Container{
		DB:            db,
		Redis:         redisClient,
		HTTPHandler:   h,
		WSClient:      ws,
		MarketService: marketSvc,
	}, nil
}

func (c *Container) Close() error {
	var firstErr error
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close redis: %w", err)
		}
	}
	if c.DB != nil {
		ch := make(chan error, 1)
		go func() { ch <- c.DB.Close() }()
		select {
		case err := <-ch:
			if err != nil && firstErr == nil {
				firstErr = fmt.Errorf("close db: %w", err)
			}
		case <-time.After(3 * time.Second):
			if firstErr == nil {
				firstErr = fmt.Errorf("close db timeout")
			}
		}
	}
	return firstErr
}
