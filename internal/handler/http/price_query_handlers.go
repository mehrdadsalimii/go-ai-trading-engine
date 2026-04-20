package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-ai-trading-engine/internal/domain"
)

func (h *Handler) latestPrice(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	if symbol == "" {
		symbol = "BTCUSDT"
	}

	price, err := h.market.GetLatestPrice(r.Context(), symbol)
	if err != nil {
		h.log.Warn("latest price not available", "error", err, "symbol", symbol)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, price)
}

type queryRequest struct {
	UserID  string `json:"user_id"`
	Symbol  string `json:"symbol"`
	Message string `json:"message"`
}

func (h *Handler) query(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	resp, err := h.ai.AnswerQuery(r.Context(), domain.UserQuery{
		UserID:  req.UserID,
		Symbol:  req.Symbol,
		Message: req.Message,
	})
	if err != nil {
		h.log.Error("query failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
