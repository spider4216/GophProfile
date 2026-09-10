package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/repositories"
)

type Service struct {
	repo   repositories.RepositoryInterface
	logger *slog.Logger
	queue  *queue.Queue
}

func New(repo repositories.RepositoryInterface, logger *slog.Logger, queue *queue.Queue) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		queue:  queue,
	}
}

func (s *Service) IsDBOK(ctx context.Context) bool {
	return s.repo.Ping(ctx) == nil
}

func (s *Service) SendUploadEvent(ctx context.Context, userID string) error {
	e := models.AvatarUploadEvent{
		AvatarID: uuid.New().String(),
		UserID:   userID,
		S3Key:    uuid.New().String(),
	}

	return s.queue.SendUploadEvent(ctx, e)
}
