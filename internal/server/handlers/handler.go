package handlers

import (
	"context"
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
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/server/models"
	"github.com/spider4216/GophProfile/internal/services"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, trace.Span)
}

type Handler struct {
	cfg     *config.Config
	logger  *slog.Logger
	service *services.Service
	tracer  Tracer
}

func New(cfg *config.Config, logger *slog.Logger, service *services.Service, tracer Tracer) Handler {
	return Handler{
		cfg:     cfg,
		logger:  logger,
		service: service,
		tracer:  tracer,
	}
}

// UploadAvatar godoc
// @Summary      Загрузить аватар
// @Description  Загружает изображение аватара пользователя. Файл временно сохраняется на сервере, после чего создаётся запись об аватаре и отправляется событие на загрузку в MinIO.
// @Tags         Avatars
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-User-ID header string true "Идентификатор пользователя"
// @Param        image formData file true "Файл изображения аватара"
// @Success      201 {object} models.UploadResp "Аватар создан, загрузка в процессе"
// @Failure      400 "Не передан файл или неподдерживаемый формат изображения"
// @Failure      413 "Размер файла превышает допустимый"
// @Failure      500 "Внутренняя ошибка сервера"
// @Router       /avatars [post]
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "UploadAvatar")
	defer span.End()
	span.SetAttributes(attribute.String("user_id", h.service.GetUserIdFromCtx(ctx)))
	sc := trace.SpanContextFromContext(ctx)

	h.logger.Debug("Upload avatar", "trace_id", sc.TraceID().String())

	file, header, err := r.FormFile("image")
	if err != nil {
		h.logger.Debug("something wrong with file", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			h.logger.Warn("cannot file close", "error", err, "trace_id", sc.TraceID().String())
		}
	}()

	fileName := header.Filename
	fileSize := header.Size
	mimetype := header.Header.Get("Content-Type")

	if !slices.Contains(h.cfg.SupportImgExt, mimetype) {
		h.logger.Debug("file format is not valid", "provided", mimetype, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if fileSize > h.cfg.MaxImgSize {
		h.logger.Debug("file is too large", "provided", fileSize, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	uid := uuid.NewString()

	// Сохраняем во временной tmp, поскольку в minio будет загружать потребитель
	if err := h.service.CreateTmpFile(ctx, fileName, file, uid); err != nil {
		h.logger.Error("cannot put file to tmp", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ava, err := h.service.CreateAvatar(ctx, fileName, mimetype, fileSize, uid)
	if err != nil {
		h.logger.Error("cannot create avatar", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Data", "filename", fileName, "size", fileSize, "mimetype", mimetype, "ID", ava.ID, "trace_id", sc.TraceID().String())
	var confirmErr queue.NoConfirmErr

	for {
		h.logger.Debug("Send upload event", "trace_id", sc.TraceID().String())
		err := h.service.SendUploadEvent(ctx, h.service.GetUserIdFromCtx(ctx), ava.ID, ava.S3Key)

		// Если нет ошибки, то выходим
		if err == nil {
			break
		}

		// Если возникла ошибка не связанная с подтверждением, то выходим
		// иначе это означает, что ошибка связана с не подтверждением
		// приема сообщения брокером, то делается retry,
		// т.е. производится повторная отправка
		if !errors.As(err, &confirmErr) {
			h.logger.Error("cannot send upload event", "error", err, "trace_id", sc.TraceID().String())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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
		h.logger.Error("cannot marshal response", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
		return
	}
}

// GetAvatar godoc
// @Summary      Получить аватар
// @Description  Возвращает изображение аватара или его миниатюру. Размер миниатюры задаётся параметром size.
// @Tags         Avatars
// @Produce      image/*
// @Param        avatar_id path string true "UUID аватара"
// @Param        size query string false "Размер миниатюры, например 100x100"
// @Success      200 {file} file "Файл изображения"
// @Failure      404 {object} models.GetNoAvaResp "Аватар не найден"
// @Failure      500 "Внутренняя ошибка сервера"
// @Header       200 {string} Content-Disposition "Имя файла"
// @Header       200 {string} Cache-Control "Параметры кеширования"
// @Header       200 {string} ETag "Хеш содержимого"
// @Router       /avatars/{avatar_id} [get]
func (h *Handler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "GetAvatar")
	defer span.End()
	span.SetAttributes(attribute.String("user_id", h.service.GetUserIdFromCtx(ctx)))
	sc := trace.SpanContextFromContext(ctx)

	id := r.PathValue("avatar_id")

	size := r.URL.Query().Get("size")

	ava, err := h.service.GetAvatarByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("avatar not found", "error", err, "trace_id", sc.TraceID().String())

			b, err := h.service.PrepareNotFoundResp()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.logger.Error("cannot marshal 404 resp", "trace_id", sc.TraceID().String())
				return
			}

			w.WriteHeader(http.StatusNotFound)

			if _, err := w.Write(b); err != nil {
				h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
				return
			}

			return
		}

		h.logger.Error("cannot get avatar", "error", err, "trace_id", sc.TraceID().String())
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
				h.logger.Error("cannot prepare not found resp", "trace_id", sc.TraceID().String())
				return
			}

			w.WriteHeader(code)

			if _, err := w.Write(b); err != nil {
				h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
				return
			}

			return
		}

		// Иначе возаращаем внутреннюю ошибку без тела
		h.logger.Error("getting avatar error", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ava.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ava.FileName))
	w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(h.cfg.CacheTTL))
	w.Header().Set("ETag", h.service.HashBinary(b))

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
		return
	}
}

// GetMetaAvatar godoc
// @Summary      Получить метаданные аватара
// @Description  Возвращает метаданные аватара по его идентификатору.
// @Tags         Avatars
// @Produce      json
// @Param        avatar_id path string true "UUID аватара"
// @Success      200 {object} models.GetMetaResp "Метаданные аватара"
// @Failure      404 "Аватар не найден"
// @Failure      500 "Внутренняя ошибка сервера"
// @Router       /avatars/{avatar_id}/metadata [get]
func (h *Handler) GetMetaAvatar(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "GetMetadataAvatar")
	defer span.End()
	span.SetAttributes(attribute.String("user_id", h.service.GetUserIdFromCtx(ctx)))
	sc := trace.SpanContextFromContext(ctx)

	id := r.PathValue("avatar_id")

	ava, err := h.service.GetAvatarByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("avatar not found", "error", err, "trace_id", sc.TraceID().String())
			w.WriteHeader(http.StatusNotFound)
			return
		}

		h.logger.Error("cannot get avatar", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := h.mapMetaResp(ava, h.cfg.ServerAddress)

	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("cannot marshal", "error", err, "trace_id", sc.TraceID().String())
		return
	}

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
		return
	}
}

// GetUserAvatar godoc
// @Summary      Получить аватар пользователя
// @Description  Возвращает последнее активное изображение пользователя или его миниатюру.
// @Tags         Users, Avatars
// @Produce      image/*
// @Param        user_id path string true "Идентификатор пользователя"
// @Param        size query string false "Размер миниатюры, например 100x100"
// @Success      200 {file} file "Файл изображения"
// @Failure      404 "Аватар пользователя не найден"
// @Failure      500 "Внутренняя ошибка сервера"
// @Header       200 {string} Content-Disposition "Имя файла"
// @Header       200 {string} Cache-Control "Параметры кеширования"
// @Header       200 {string} ETag "Хеш содержимого"
// @Router       /users/{user_id}/avatar [get]
func (h *Handler) GetUserAvatar(w http.ResponseWriter, r *http.Request) {
	size := r.URL.Query().Get("size")
	userID := r.PathValue("user_id")

	ctx, span := h.tracer.Start(r.Context(), "GetUserAvatar")
	defer span.End()
	span.SetAttributes(attribute.String("user_id", userID), attribute.String("size", size))
	sc := trace.SpanContextFromContext(ctx)

	ava, err := h.service.GetLatestActiveUserAvatar(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("avatar not found", "error", err, "trace_id", sc.TraceID().String())
			w.WriteHeader(http.StatusNotFound)
			return
		}

		h.logger.Error("cannot get avatar", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Получаем аватар или thumbnail
	b, code, err := h.service.GetComplexBinaryAva(ctx, size, ava)
	if err != nil {
		// Если аватара нет, то возвращаем ответ
		if code == http.StatusNotFound {
			h.logger.Debug("avatar binary not found", "error", err, "trace_id", sc.TraceID().String())
			w.WriteHeader(code)
			return
		}

		// Иначе возаращаем внутреннюю ошибку
		h.logger.Error("getting avatar binary error", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ava.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ava.FileName))
	w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(h.cfg.CacheTTL))
	w.Header().Set("ETag", h.service.HashBinary(b))

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
		return
	}
}

// DeleteAvatar godoc
// @Summary      Удалить аватар
// @Description  Запускает удаление аватара пользователя через отправку события в брокер сообщений. Удалять можно только собственный аватар.
// @Tags         Avatars
// @Param        id path string true "UUID аватара"
// @Param        X-User-ID header string true "Идентификатор пользователя"
// @Success      204 "Аватар успешно помечен для удаления"
// @Failure      403 {object} models.ForbiddenResp "Пользователь не является владельцем аватара"
// @Failure      404 "Аватар не найден"
// @Failure      500 "Внутренняя ошибка сервера"
// @Router       /avatars/{id} [delete]
func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "DeleteAvatar")
	defer span.End()
	span.SetAttributes(attribute.String("user_id", h.service.GetUserIdFromCtx(ctx)))
	sc := trace.SpanContextFromContext(ctx)

	userID := h.service.GetUserIdFromCtx(ctx)

	avaID := r.PathValue("id")

	ava, err := h.service.GetAvatarByID(ctx, avaID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("avatar not found", "error", err, "trace_id", sc.TraceID().String())
			w.WriteHeader(http.StatusNotFound)
			return
		}

		h.logger.Error("cannot get avatar", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if ava.UserID != userID {
		h.logger.Error("ava user id not match with request user id", "trace_id", sc.TraceID().String())

		b, err := h.service.PrepareForbiddenResp()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.logger.Error("cannot prepare forbidden resp", "error", err, "trace_id", sc.TraceID().String())
			return
		}

		w.WriteHeader(http.StatusForbidden)

		if _, err := w.Write(b); err != nil {
			h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
			return
		}

		return
	}

	var confirmErr queue.NoConfirmErr

	for {
		err := h.service.SendDeleteEvent(ctx, ava.ID)

		// Если нет ошибки, то выходим
		if err == nil {
			break
		}

		// Если возникла ошибка не связанная с подтверждением, то выходим
		// иначе это означает, что ошибка связана с не подтверждением
		// приема сообщения брокером, то делается retry,
		// т.е. производится повторная отправка
		if !errors.As(err, &confirmErr) {
			h.logger.Error("cannot send delete event", "error", err, "trace_id", sc.TraceID().String())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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

// DeleteUserAvatars godoc
// @Summary      Удалить все аватары пользователя
// @Description  Запускает асинхронное удаление всех аватаров указанного пользователя через отправку событий в брокер сообщений. Удалять аватары можно только от имени их владельца.
// @Tags         Users, Avatars
// @Param        user_id path string true "Идентификатор пользователя"
// @Param        X-User-ID header string true "Идентификатор текущего пользователя"
// @Success      204 "События удаления успешно отправлены"
// @Failure      403 {object} models.ForbiddenResp "Текущий пользователь не является владельцем всех аватаров"
// @Failure      404 "У пользователя нет аватаров"
// @Failure      500 "Внутренняя ошибка сервера"
// @Router       /users/{user_id}/avatar [delete]
func (h *Handler) DeleteUserAvatars(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")

	ctx, span := h.tracer.Start(r.Context(), "DeleteUserAvatars")
	defer span.End()
	span.SetAttributes(
		attribute.String("current_user_id", h.service.GetUserIdFromCtx(ctx)),
		attribute.String("user_id", h.service.GetUserIdFromCtx(ctx)),
	)
	sc := trace.SpanContextFromContext(ctx)

	avas, err := h.service.GetUserAvatars(ctx, userID)
	if err != nil {
		h.logger.Error("cannot get user avatars", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(avas) <= 0 {
		h.logger.Debug("user avatars not found", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, ava := range avas {
		if ava.UserID != h.service.GetUserIdFromCtx(ctx) {
			h.logger.Debug("ava user id not match with request user id", "trace_id", sc.TraceID().String())

			b, err := h.service.PrepareForbiddenResp()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.logger.Error("cannot prepare forbidden resp", "error", err, "trace_id", sc.TraceID().String())
				return
			}

			w.WriteHeader(http.StatusForbidden)

			if _, err := w.Write(b); err != nil {
				h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
				return
			}

			return
		}
	}

	var confirmErr queue.NoConfirmErr

	for {
		err := h.service.SendDeleteEvents(ctx, avas)

		// Если нет ошибки, то выходим
		if err == nil {
			break
		}

		// Если возникла ошибка не связанная с подтверждением, то выходим
		// иначе это означает, что ошибка связана с не подтверждением
		// приема сообщения брокером, то делается retry,
		// т.е. производится повторная отправка
		if !errors.As(err, &confirmErr) {
			h.logger.Error("cannot send delete event", "error", err, "trace_id", sc.TraceID().String())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUserAvatars godoc
// @Summary      Получить все аватары пользователя
// @Description  Возвращает список метаданных всех аватаров указанного пользователя.
// @Tags         Users, Avatars
// @Produce      json
// @Param        user_id path string true "Идентификатор пользователя"
// @Success      200 {array} models.GetMetaResp "Список метаданных аватаров"
// @Failure      404 "У пользователя нет аватаров"
// @Failure      500 "Внутренняя ошибка сервера"
// @Router       /users/{user_id}/avatars [get]
func (h *Handler) GetUserAvatars(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	ctx, span := h.tracer.Start(r.Context(), "GetUserAvatars")
	defer span.End()
	span.SetAttributes(attribute.String("user_id", userID))
	sc := trace.SpanContextFromContext(ctx)

	avas, err := h.service.GetUserAvatars(ctx, userID)
	if err != nil {
		h.logger.Error("cannot get user avatars", "error", err, "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(avas) <= 0 {
		h.logger.Debug("user avatars not found", "trace_id", sc.TraceID().String())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp := h.mapMetasResp(avas, h.cfg.ServerAddress)

	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error("cannot marshal", "error", err, "trace_id", sc.TraceID().String())
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", "error", err, "trace_id", sc.TraceID().String())
		return
	}
}
