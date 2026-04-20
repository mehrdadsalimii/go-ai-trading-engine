package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"go-ai-trading-engine/internal/domain"
)

type TradingRepository struct {
	db *sql.DB
}

func NewTradingRepository(db *sql.DB) *TradingRepository {
	return &TradingRepository{db: db}
}

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := ensureSchema(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

func ensureSchema(ctx context.Context, db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS price_ticks (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    trade_id TEXT NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_price_ticks_symbol_event_time ON price_ticks(symbol, event_time DESC);

CREATE TABLE IF NOT EXISTS ai_query_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    latest_price DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_query_logs_symbol_created_at ON ai_query_logs(symbol, created_at DESC);
`
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	return nil
}

func (r *TradingRepository) StoreTick(ctx context.Context, tick domain.PriceTick) error {
	const q = `INSERT INTO price_ticks(symbol, price, trade_id, event_time, source) VALUES($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q, tick.Symbol, tick.Price, tick.TradeID, tick.EventTime, tick.Source)
	if err != nil {
		return fmt.Errorf("insert price tick: %w", err)
	}
	return nil
}

func (r *TradingRepository) StoreQueryLog(ctx context.Context, log domain.AIQueryLog) error {
	const q = `INSERT INTO ai_query_logs(user_id, symbol, question, answer, latest_price, created_at) VALUES($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, q, log.UserID, log.Symbol, log.Question, log.Answer, log.LatestPrice, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert query log: %w", err)
	}
	return nil
}
