package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv               string
	HTTPPort             string
	ShutdownTimeout      time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	PostgresDSN          string
	OpenAIAPIKey         string
	OpenAIModel          string
	BinanceWSURL         string
	PriceStreamSymbol    string
	PriceChannelBuffer   int
	RateLimitRPS         float64
	RateLimitBurst       int
	LatestPriceTTL       time.Duration
	RecentHistoryMaxSize int
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPPort:             getEnv("HTTP_PORT", "8080"),
		ShutdownTimeout:      getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:          getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:         getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:          getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		RedisDB:              getEnvInt("REDIS_DB", 0),
		PostgresDSN:          getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/trading?sslmode=disable"),
		OpenAIAPIKey:         getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:          getEnv("OPENAI_MODEL", "gpt-4.1-mini"),
		BinanceWSURL:         getEnv("BINANCE_WS_URL", "wss://stream.binance.com:9443/ws/btcusdt@trade"),
		PriceStreamSymbol:    getEnv("PRICE_STREAM_SYMBOL", "BTCUSDT"),
		PriceChannelBuffer:   getEnvInt("PRICE_CHANNEL_BUFFER", 2048),
		RateLimitRPS:         getEnvFloat("RATE_LIMIT_RPS", 5),
		RateLimitBurst:       getEnvInt("RATE_LIMIT_BURST", 10),
		LatestPriceTTL:       getEnvDuration("LATEST_PRICE_TTL", 10*time.Minute),
		RecentHistoryMaxSize: getEnvInt("RECENT_HISTORY_MAX_SIZE", 200),
	}

	if cfg.OpenAIAPIKey == "" {
		return Config{}, fmt.Errorf("OPENAI_API_KEY is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
