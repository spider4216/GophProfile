package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/spider4216/GophProfile/internal/server/config"
	"github.com/spider4216/GophProfile/internal/server/models"
	"github.com/spider4216/GophProfile/internal/services"
)

type Handler struct {
	cfg     *config.Config
	logger  *slog.Logger
	service *services.Service
}

func New(cfg *config.Config, logger *slog.Logger, service *services.Service) Handler {
	return Handler{
		cfg:     cfg,
		logger:  logger,
		service: service,
	}
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	// todo logic
}

func (h *Handler) GetUserAvatar(w http.ResponseWriter, r *http.Request) {
	// todo logic
}

func (h *Handler) GetMetaAvatars(w http.ResponseWriter, r *http.Request) {
	// todo logic
}

func (h *Handler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	// todo logic
}

func (h *Handler) DeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	// todo logic
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// todo logic
	resp := models.HealthResp{
		DB: h.service.IsDBOK(ctx),
	}

	b, err := json.Marshal(resp)

	if err != nil {
		h.logger.Error("cannot marshal response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err)
		return
	}
}
