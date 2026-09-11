package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"

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
	ctx := r.Context()
	// todo validate filesize

	// todo field name to const
	file, header, err := r.FormFile("file")

	if err != nil {
		h.logger.Error("something wrong with file", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer file.Close()

	fileName := header.Filename
	fileSize := header.Size
	mimetype := header.Header.Get("Content-Type")

	if !slices.Contains(h.cfg.SupportImgExt, mimetype) {
		h.logger.Error("file is not valid", "provided", mimetype)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Debug("Data", "filename", fileName, "size", fileSize, "mimetype", mimetype)

	if err := h.service.SendUploadEvent(ctx, "test_user_id"); err != nil {
		h.logger.Error("cannot send upload event", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
