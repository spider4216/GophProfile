package middlewares

import (
	"log/slog"

	"github.com/spider4216/GophProfile/internal/server/config"
)

type Middleware struct {
	logger *slog.Logger
	cfg    *config.Config
}

func New(logger *slog.Logger, cfg *config.Config) Middleware {
	return Middleware{
		logger: logger,
		cfg:    cfg,
	}
}
