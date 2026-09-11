package handlers

import (
	"context"
	"log/slog"

	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/worker/services"
)

type Handler struct {
	logger  *slog.Logger
	service *services.Service
}

func NewHandler(logger *slog.Logger, service *services.Service) *Handler {
	return &Handler{
		logger:  logger,
		service: service,
	}
}

func (h *Handler) UploadAvatar(ctx context.Context, e models.AvatarUploadEvent) error {
	h.logger.Debug("Event avatar", "ID", e.AvatarID, "user", e.UserID, "s3key", e.S3Key)
	h.service.Upload()
	return nil
}
