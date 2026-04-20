package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"go-ai-trading-engine/internal/domain"
)

type BinanceClient struct {
	url string
	log *slog.Logger
}

func NewBinanceClient(url string, logger *slog.Logger) *BinanceClient {
	return &BinanceClient{url: url, log: logger}
}

type tradeMsg struct {
	TradeID int64  `json:"t"`
	Symbol  string `json:"s"`
	Price   string `json:"p"`
}

func (c *BinanceClient) Start(ctx context.Context, out chan<- domain.PriceTick) {
	go c.run(ctx, out)
}

func (c *BinanceClient) run(ctx context.Context, out chan<- domain.PriceTick) {
	backoff := 1 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := c.streamOnce(ctx, out); err != nil {
			c.log.Error("binance stream disconnected", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (c *BinanceClient) streamOnce(ctx context.Context, out chan<- domain.PriceTick) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()

	c.log.Info("connected to binance stream", "url", c.url)
	if err := conn.SetReadDeadline(time.Now().Add(45 * time.Second)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(45 * time.Second))
	})

	pingTicker := time.NewTicker(20 * time.Second)
	defer pingTicker.Stop()

	errCh := make(chan error, 1)
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				errCh <- fmt.Errorf("read message: %w", err)
				return
			}

			msg, err := parseTradeMessage(message)
			if err != nil {
				c.log.Warn("skip malformed trade message", "error", err)
				continue
			}
			price, err := strconv.ParseFloat(msg.Price, 64)
			if err != nil {
				c.log.Warn("skip trade with invalid price", "error", err, "price", msg.Price)
				continue
			}

			tick := domain.PriceTick{
				Symbol:    strings.ToUpper(msg.Symbol),
				Price:     price,
				TradeID:   strconv.FormatInt(int64(msg.TradeID), 10),
				EventTime: time.UnixMilli(time.Now().UnixMilli()).UTC(),
				Source:    "binance_ws",
			}

			select {
			case out <- tick:
			default:
				c.log.Warn("tick channel full, dropping tick", "symbol", tick.Symbol)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return fmt.Errorf("send ping: %w", err)
			}
		}
	}
}

func parseTradeMessage(message []byte) (tradeMsg, error) {
	var msg tradeMsg
	if err := json.Unmarshal(message, &msg); err != nil {
		return tradeMsg{}, fmt.Errorf("unmarshal trade message: %w", err)
	}
	if msg.Symbol == "" {
		return tradeMsg{}, fmt.Errorf("unmarshal trade message: missing symbol")
	}
	return msg, nil
}
