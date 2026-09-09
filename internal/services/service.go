package services

import (
	"context"
	"log/slog"

	"github.com/spider4216/GophProfile/internal/repositories"
)

type Service struct {
	repo   repositories.RepositoryInterface
	logger *slog.Logger
}

func New(repo repositories.RepositoryInterface, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) IsDBOK(ctx context.Context) bool {
	return s.repo.Ping(ctx) == nil
}
