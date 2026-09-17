package logger

import (
	"log/slog"
	"os"
	"strings"
)

func Init(lvl string) *slog.Logger {
	return slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: parseLogLevel(lvl),
		}),
	)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
