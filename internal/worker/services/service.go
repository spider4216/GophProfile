package services

import (
	"log/slog"

	"github.com/spider4216/GophProfile/internal/queue"
)

type Service struct {
	logger *slog.Logger
	queue  *queue.Queue
}

func NewService(logger *slog.Logger, q *queue.Queue) *Service {
	return &Service{
		logger: logger,
		queue:  q,
	}
}

func (s *Service) Upload() {
	s.logger.Debug("Service upload")
}
