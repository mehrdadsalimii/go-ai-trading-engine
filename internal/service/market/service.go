package market

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"go-ai-trading-engine/internal/domain"
)

type Service struct {
	priceRepo      domain.PriceRepository
	tradingRepo    domain.TradingRepository
	log            *slog.Logger
	recentHistoryN int
}

func NewService(priceRepo domain.PriceRepository, tradingRepo domain.TradingRepository, logger *slog.Logger, recentHistoryN int) *Service {
	return &Service{
		priceRepo:      priceRepo,
		tradingRepo:    tradingRepo,
		log:            logger,
		recentHistoryN: recentHistoryN,
	}
}

func (s *Service) StartTickProcessors(ctx context.Context, ticks <-chan domain.PriceTick, workers int) {
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.log.Info("tick processor started", "worker", workerID)
			for {
				select {
				case <-ctx.Done():
					s.log.Info("tick processor stopping", "worker", workerID)
					return
				case tick, ok := <-ticks:
					if !ok {
						return
					}
					if err := s.ProcessTick(ctx, tick); err != nil {
						s.log.Error("process tick failed", "error", err, "symbol", tick.Symbol)
					}
				}
			}
		}(i + 1)
	}

	go func() {
		<-ctx.Done()
		wg.Wait()
		s.log.Info("all tick processors stopped")
	}()
}

func (s *Service) ProcessTick(ctx context.Context, tick domain.PriceTick) error {
	tick.Symbol = strings.ToUpper(tick.Symbol)
	if err := s.priceRepo.SetLatestPrice(ctx, tick); err != nil {
		return fmt.Errorf("set latest price: %w", err)
	}
	if err := s.priceRepo.PushRecentPrice(ctx, tick, s.recentHistoryN); err != nil {
		return fmt.Errorf("push recent price: %w", err)
	}
	if err := s.tradingRepo.StoreTick(ctx, tick); err != nil {
		return fmt.Errorf("store tick in db: %w", err)
	}
	return nil
}

func (s *Service) GetLatestPrice(ctx context.Context, symbol string) (domain.PriceTick, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return domain.PriceTick{}, fmt.Errorf("symbol is required")
	}
	return s.priceRepo.GetLatestPrice(ctx, symbol)
}

func (s *Service) GetRecentPrices(ctx context.Context, symbol string, limit int64) ([]domain.PriceTick, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.priceRepo.GetRecentPrices(ctx, symbol, limit)
}
