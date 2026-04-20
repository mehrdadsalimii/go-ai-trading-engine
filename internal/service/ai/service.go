package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go-ai-trading-engine/internal/domain"
)

type marketReader interface {
	GetLatestPrice(ctx context.Context, symbol string) (domain.PriceTick, error)
	GetRecentPrices(ctx context.Context, symbol string, limit int64) ([]domain.PriceTick, error)
}

type Service struct {
	aiClient      domain.AIClient
	tradingRepo   domain.TradingRepository
	marketService marketReader
	log           *slog.Logger
}

func NewService(aiClient domain.AIClient, tradingRepo domain.TradingRepository, marketService marketReader, logger *slog.Logger) *Service {
	return &Service{
		aiClient:      aiClient,
		tradingRepo:   tradingRepo,
		marketService: marketService,
		log:           logger,
	}
}

func (s *Service) AnswerQuery(ctx context.Context, query domain.UserQuery) (domain.QueryResponse, error) {
	query.Symbol = strings.ToUpper(strings.TrimSpace(query.Symbol))
	if query.Symbol == "" {
		query.Symbol = "BTCUSDT"
	}
	if strings.TrimSpace(query.Message) == "" {
		return domain.QueryResponse{}, fmt.Errorf("query message is required")
	}

	latest, err := s.marketService.GetLatestPrice(ctx, query.Symbol)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("fetch latest price: %w", err)
	}
	recent, err := s.marketService.GetRecentPrices(ctx, query.Symbol, 20)
	if err != nil {
		s.log.Warn("failed to fetch recent prices", "error", err)
	}

	prompt := buildPrompt(query, latest, recent)
	answer, err := s.aiClient.Analyze(ctx, prompt)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("ai analyze: %w", err)
	}

	now := time.Now().UTC()
	if err := s.tradingRepo.StoreQueryLog(ctx, domain.AIQueryLog{
		UserID:      query.UserID,
		Symbol:      query.Symbol,
		Question:    query.Message,
		Answer:      answer,
		LatestPrice: latest.Price,
		CreatedAt:   now,
	}); err != nil {
		s.log.Error("failed to store query log", "error", err)
	}

	return domain.QueryResponse{
		Answer:      answer,
		GeneratedAt: now,
	}, nil
}

func buildPrompt(query domain.UserQuery, latest domain.PriceTick, recent []domain.PriceTick) string {
	recentPart := "No recent trades available"
	if len(recent) > 0 {
		start := recent[len(recent)-1]
		end := recent[0]
		trend := "flat"
		if end.Price > start.Price {
			trend = "up"
		} else if end.Price < start.Price {
			trend = "down"
		}
		recentPart = fmt.Sprintf("Recent ticks analyzed: %d, start=%.2f, latest=%.2f, short-trend=%s", len(recent), start.Price, end.Price, trend)
	}

	return fmt.Sprintf(`User question: %s
Symbol: %s
Latest price: %.2f at %s
%s
Respond with:
1) directional bias (bullish/bearish/neutral) with confidence 0-100
2) concise reasoning
3) one risk warning
4) this is not financial advice`,
		query.Message,
		query.Symbol,
		latest.Price,
		latest.EventTime.Format(time.RFC3339),
		recentPart,
	)
}
