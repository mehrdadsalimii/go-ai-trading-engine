package domain

import "time"

type PriceTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	TradeID   string    `json:"trade_id"`
	EventTime time.Time `json:"event_time"`
	Source    string    `json:"source"`
}

type UserQuery struct {
	UserID  string `json:"user_id"`
	Symbol  string `json:"symbol"`
	Message string `json:"message"`
}

type QueryResponse struct {
	Answer      string    `json:"answer"`
	GeneratedAt time.Time `json:"generated_at"`
}

type AIQueryLog struct {
	UserID      string
	Symbol      string
	Question    string
	Answer      string
	CreatedAt   time.Time
	LatestPrice float64
}
