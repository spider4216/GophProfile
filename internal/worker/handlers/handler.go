package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/worker/config"
	"github.com/spider4216/GophProfile/internal/worker/services"
)

type Handler struct {
	logger  *slog.Logger
	service *services.Service
	cfg     *config.Config
}

func NewHandler(logger *slog.Logger, service *services.Service, cfg *config.Config) *Handler {
	return &Handler{
		logger:  logger,
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) UploadAvatar(ctx context.Context, e models.AvatarUploadEvent) error {
	h.logger.Debug("Event avatar", "ID", e.AvatarID, "user", e.UserID, "s3key", e.S3Key)

	h.logger.Debug("upload avatar to minio", "avatar", e.AvatarID)

	err := h.service.Upload(ctx, e)

	if err != nil {
		return fmt.Errorf("cannot upload avatar: %w", err)
	}

	return h.service.SendProcessEvent(ctx, e.AvatarID)
}

func (h *Handler) ProcessAvatar(ctx context.Context, e models.AvatarProcessEvent) error {
	return h.service.ProcessAvatar(ctx, e, h.cfg.QualityProcess)
}
