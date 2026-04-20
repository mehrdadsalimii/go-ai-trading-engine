package logger

import (
	"log/slog"
	"os"
)

func New(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}
	h := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(h)
}
