package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"go-ai-trading-engine/internal/domain"
)

type marketService interface {
	GetLatestPrice(ctx context.Context, symbol string) (domain.PriceTick, error)
}

type aiService interface {
	AnswerQuery(ctx context.Context, query domain.UserQuery) (domain.QueryResponse, error)
}

type Handler struct {
	market marketService
	ai     aiService
	log    *slog.Logger
}

func NewHandler(market marketService, ai aiService, logger *slog.Logger) *Handler {
	return &Handler{market: market, ai: ai, log: logger}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/v1/prices/latest", h.latestPrice)
	mux.HandleFunc("POST /api/v1/query", h.query)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
