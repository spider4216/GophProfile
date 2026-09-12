package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/spider4216/GophProfile/internal/enum"
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
		h.logger.Error("file format is not valid", "provided", mimetype)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if fileSize > h.cfg.MaxImgSize {
		h.logger.Error("file is too large", "provided", fileSize)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	uid := uuid.NewString()

	// Сохраняем во временной tmp, поскольку в minio будет загружать потребитель
	if err := h.service.CreateTmpFile(ctx, fileName, file, uid); err != nil {
		h.logger.Error("cannot put file to tmp", "error", err)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	ava, err := h.service.CreateAvatar(ctx, fileName, mimetype, fileSize, uid)

	if err != nil {
		h.logger.Error("cannot create avatar", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Data", "filename", fileName, "size", fileSize, "mimetype", mimetype, "ID", ava.ID)

	if err := h.service.SendUploadEvent(ctx, h.service.GetUserIdFromCtx(ctx), ava.ID, ava.S3Key); err != nil {
		h.logger.Error("cannot send upload event", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	url := path.Join("https://", h.cfg.ServerAddress, "/api/v1/avatars/", ava.ID)

	resp := models.UploadResp{
		ID:        ava.ID,
		UserID:    ava.UserID,
		URL:       url,
		Status:    enum.Uploading,
		CreatedAt: time.Now(),
	}

	b, err := json.Marshal(resp)

	if err != nil {
		h.logger.Error("cannot marshal response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err)
		return
	}
}

func (h *Handler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("avatar_id")

	size := r.URL.Query().Get("size")

	ava, err := h.service.GetAvatarByID(ctx, id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Error("avatar not found", "error", err)

			b, err := h.service.PrepareNotFoundResp()

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.logger.Error("cannot marshal 404 resp")
				return
			}

			w.WriteHeader(http.StatusNotFound)

			if _, err := w.Write(b); err != nil {
				h.logger.Error("failed to write response", "error", err)
				return
			}

			return
		}

		h.logger.Error("cannot get avatar", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var b []byte

	// Получаем аватар или thumbnail
	b, code, err := h.service.GetComplexBinaryAva(ctx, size, ava)

	if err != nil {
		// Если аватара нет, то возвращаем ответ с телом
		if code == http.StatusNotFound {
			b, err := h.service.PrepareNotFoundResp()

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.logger.Error("cannot marshal 404 resp")
				return
			}

			w.WriteHeader(code)

			if _, err := w.Write(b); err != nil {
				h.logger.Error("failed to write response", "error", err)
				return
			}

			return
		}

		// Иначе возаращаем внутреннюю ошибку без тела
		h.logger.Error("getting avatar error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ava.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ava.FileName))
	w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(h.cfg.CacheTTL))
	w.Header().Set("ETag", h.service.HashBinary(b))

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err)
		return
	}
}

func (h *Handler) GetMetaAvatars(w http.ResponseWriter, r *http.Request) {
	// todo logic
}

func (h *Handler) GetUserAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	size := r.URL.Query().Get("size")

	userID := r.PathValue("user_id")

	ava, err := h.service.GetLatestActiveUserAvatar(ctx, userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Error("avatar not found", "error", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		h.logger.Error("cannot get avatar", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Получаем аватар или thumbnail
	b, code, err := h.service.GetComplexBinaryAva(ctx, size, ava)

	if err != nil {
		// Если аватара нет, то возвращаем ответ
		if code == http.StatusNotFound {
			h.logger.Error("avatar binary not found", "error", err)
			w.WriteHeader(code)
			return
		}

		// Иначе возаращаем внутреннюю ошибку
		h.logger.Error("getting avatar binary error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ava.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ava.FileName))
	w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(h.cfg.CacheTTL))
	w.Header().Set("ETag", h.service.HashBinary(b))

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err)
		return
	}
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
