package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/spider4216/GophProfile/internal/config"
	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/worker/services"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, trace.Span)
}

type Handler struct {
	logger  *slog.Logger
	service *services.Service
	cfg     *config.Config
	tracer  Tracer
}

func NewHandler(logger *slog.Logger, service *services.Service, cfg *config.Config, tracer Tracer) *Handler {
	return &Handler{
		logger:  logger,
		service: service,
		cfg:     cfg,
		tracer:  tracer,
	}
}

func (h *Handler) UploadAvatar(ctx context.Context, e models.AvatarUploadEvent) error {
	ctx, span := h.tracer.Start(ctx, "UploadAvatarConsume")
	defer span.End()
	span.SetAttributes(attribute.String("avatarID", e.AvatarID), attribute.String("user_id", e.UserID))

	err := h.service.Upload(ctx, e)
	if err != nil {
		return fmt.Errorf("cannot upload avatar: %w", err)
	}

	var confirmErr queue.NoConfirmErr

	for {
		err := h.service.SendProcessEvent(ctx, e.AvatarID)
		// Если нет ошибки, то выходим
		if err != nil {
			break
		}

		// Если возникла ошибка не связанная с подтверждением, то выходим
		// иначе это означает, что ошибка связана с не подтверждением
		// приема сообщения брокером, то делается retry,
		// т.е. производится повторная отправка
		if !errors.As(err, &confirmErr) {
			return err
		}
	}

	return nil
}

func (h *Handler) ProcessAvatar(ctx context.Context, e models.AvatarProcessEvent) error {
	ctx, span := h.tracer.Start(ctx, "ProcessAvatarConsume")
	defer span.End()
	span.SetAttributes(attribute.String("avatarID", e.AvatarID))

	return h.service.ProcessAvatar(ctx, e, h.cfg.QualityProcess)
}

func (h *Handler) DeleteAvatar(ctx context.Context, e *models.AvatarDeleteEvent) error {
	ctx, span := h.tracer.Start(ctx, "DeleteAvatarConsume")
	defer span.End()
	span.SetAttributes(attribute.String("avatarID", e.AvatarID))

	return h.service.DeleteAvatar(ctx, e)
}
