package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"go-ai-trading-engine/internal/domain"
)

type PriceRepository struct {
	client *redis.Client
	ttl    int64
}

func NewPriceRepository(client *redis.Client, ttlSeconds int64) *PriceRepository {
	return &PriceRepository{client: client, ttl: ttlSeconds}
}

func (r *PriceRepository) latestKey(symbol string) string {
	return fmt.Sprintf("price:latest:%s", strings.ToUpper(symbol))
}

func (r *PriceRepository) historyKey(symbol string) string {
	return fmt.Sprintf("price:history:%s", strings.ToUpper(symbol))
}

func (r *PriceRepository) SetLatestPrice(ctx context.Context, tick domain.PriceTick) error {
	b, err := json.Marshal(tick)
	if err != nil {
		return fmt.Errorf("marshal tick: %w", err)
	}
	ttl := time.Duration(r.ttl) * time.Second
	return r.client.Set(ctx, r.latestKey(tick.Symbol), b, ttl).Err()
}

func (r *PriceRepository) GetLatestPrice(ctx context.Context, symbol string) (domain.PriceTick, error) {
	val, err := r.client.Get(ctx, r.latestKey(symbol)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.PriceTick{}, fmt.Errorf("latest price not found for %s", symbol)
		}
		return domain.PriceTick{}, fmt.Errorf("redis get latest price: %w", err)
	}

	var tick domain.PriceTick
	if err := json.Unmarshal([]byte(val), &tick); err != nil {
		return domain.PriceTick{}, fmt.Errorf("unmarshal tick: %w", err)
	}
	return tick, nil
}

func (r *PriceRepository) PushRecentPrice(ctx context.Context, tick domain.PriceTick, maxSize int) error {
	b, err := json.Marshal(tick)
	if err != nil {
		return fmt.Errorf("marshal tick: %w", err)
	}
	pipe := r.client.TxPipeline()
	pipe.LPush(ctx, r.historyKey(tick.Symbol), b)
	pipe.LTrim(ctx, r.historyKey(tick.Symbol), 0, int64(maxSize-1))
	pipe.Expire(ctx, r.historyKey(tick.Symbol), time.Duration(r.ttl)*time.Second)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("push recent price: %w", err)
	}
	return nil
}

func (r *PriceRepository) GetRecentPrices(ctx context.Context, symbol string, limit int64) ([]domain.PriceTick, error) {
	rows, err := r.client.LRange(ctx, r.historyKey(symbol), 0, limit-1).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("redis lrange recent prices: %w", err)
	}

	out := make([]domain.PriceTick, 0, len(rows))
	for _, row := range rows {
		var tick domain.PriceTick
		if err := json.Unmarshal([]byte(row), &tick); err != nil {
			continue
		}
		out = append(out, tick)
	}
	return out, nil
}
