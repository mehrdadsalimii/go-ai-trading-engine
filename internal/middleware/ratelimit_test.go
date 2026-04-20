package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitBlocksAfterBurst(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(1, 1)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request expected 200, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request expected 429, got %d", w2.Code)
	}
}
