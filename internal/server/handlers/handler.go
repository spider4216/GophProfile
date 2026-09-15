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
	"github.com/spider4216/GophProfile/internal/config"
	"github.com/spider4216/GophProfile/internal/enum"
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

	file, header, err := r.FormFile("image")
	if err != nil {
		h.logger.Error("something wrong with file", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			h.logger.Warn("cannot file close", "error", err)
		}
	}()

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

func (h *Handler) GetMetaAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("avatar_id")

	ava, err := h.service.GetAvatarByID(ctx, id)
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

	resp := h.mapMetaResp(ava, h.cfg.ServerAddress)

	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("cannot marshal")
		return
	}

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err)
		return
	}
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

func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Delete avatar")
	ctx := r.Context()

	userID := h.service.GetUserIdFromCtx(ctx)

	avaID := r.PathValue("id")

	ava, err := h.service.GetAvatarByID(ctx, avaID)
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

	if ava.UserID != userID {
		h.logger.Error("ava user id not match with request user id")

		b, err := h.service.PrepareForbiddenResp()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.logger.Error("cannot marshal forbidden resp")
			return
		}

		w.WriteHeader(http.StatusForbidden)

		if _, err := w.Write(b); err != nil {
			h.logger.Error("failed to write response", "error", err)
			return
		}

		return
	}

	if err := h.service.SendDeleteEvent(ctx, ava.ID); err != nil {
		h.logger.Error("cannot send evet fpr delete ava", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

func (h *Handler) DeleteUserAvatars(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Delete user avatars")

	userID := r.PathValue("user_id")
	ctx := r.Context()

	avas, err := h.service.GetUserAvatars(ctx, userID)
	if err != nil {
		h.logger.Error("cannot get user avatars", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(avas) <= 0 {
		h.logger.Error("user avatars not found", "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := h.service.SendDeleteEvents(ctx, avas); err != nil {
		h.logger.Error("cannot send evet for delete avatars", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUserAvatars(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Delete user avatars")

	userID := r.PathValue("user_id")
	ctx := r.Context()

	avas, err := h.service.GetUserAvatars(ctx, userID)
	if err != nil {
		h.logger.Error("cannot get user avatars", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(avas) <= 0 {
		h.logger.Error("user avatars not found", "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp := h.mapMetasResp(avas, h.cfg.ServerAddress)

	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("cannot marshal")
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err)
		return
	}
}
