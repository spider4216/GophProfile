package middlewares

import (
	"log/slog"

	"github.com/spider4216/GophProfile/internal/config"
	"github.com/spider4216/GophProfile/internal/services"
)

type Middleware struct {
	logger  *slog.Logger
	cfg     *config.Config
	service *services.Service
}

func New(logger *slog.Logger, cfg *config.Config, service *services.Service) Middleware {
	return Middleware{
		logger:  logger,
		cfg:     cfg,
		service: service,
	}
}
