package market

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"go-ai-trading-engine/internal/domain"
)

type mockPriceRepo struct {
	latest domain.PriceTick
}

func (m *mockPriceRepo) SetLatestPrice(_ context.Context, tick domain.PriceTick) error {
	m.latest = tick
	return nil
}

func (m *mockPriceRepo) GetLatestPrice(_ context.Context, _ string) (domain.PriceTick, error) {
	if m.latest.Symbol == "" {
		return domain.PriceTick{}, errors.New("not found")
	}
	return m.latest, nil
}

func (m *mockPriceRepo) PushRecentPrice(_ context.Context, _ domain.PriceTick, _ int) error {
	return nil
}
func (m *mockPriceRepo) GetRecentPrices(_ context.Context, _ string, _ int64) ([]domain.PriceTick, error) {
	return []domain.PriceTick{m.latest}, nil
}

type mockTradingRepo struct{}

func (m *mockTradingRepo) StoreTick(_ context.Context, _ domain.PriceTick) error      { return nil }
func (m *mockTradingRepo) StoreQueryLog(_ context.Context, _ domain.AIQueryLog) error { return nil }

func TestProcessTickAndGetLatest(t *testing.T) {
	t.Parallel()

	priceRepo := &mockPriceRepo{}
	tradingRepo := &mockTradingRepo{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(priceRepo, tradingRepo, logger, 100)

	tick := domain.PriceTick{
		Symbol:    "btcusdt",
		Price:     65000.5,
		TradeID:   "1",
		EventTime: time.Now().UTC(),
		Source:    "test",
	}

	if err := svc.ProcessTick(context.Background(), tick); err != nil {
		t.Fatalf("ProcessTick() error = %v", err)
	}

	got, err := svc.GetLatestPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("GetLatestPrice() error = %v", err)
	}
	if got.Symbol != "BTCUSDT" {
		t.Fatalf("expected symbol BTCUSDT, got %s", got.Symbol)
	}
	if got.Price != tick.Price {
		t.Fatalf("expected price %.2f, got %.2f", tick.Price, got.Price)
	}
}
