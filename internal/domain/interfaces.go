package domain

import "context"

type PriceRepository interface {
	SetLatestPrice(ctx context.Context, tick PriceTick) error
	GetLatestPrice(ctx context.Context, symbol string) (PriceTick, error)
	PushRecentPrice(ctx context.Context, tick PriceTick, maxSize int) error
	GetRecentPrices(ctx context.Context, symbol string, limit int64) ([]PriceTick, error)
}

type TradingRepository interface {
	StoreTick(ctx context.Context, tick PriceTick) error
	StoreQueryLog(ctx context.Context, log AIQueryLog) error
}

type AIClient interface {
	Analyze(ctx context.Context, prompt string) (string, error)
}
