package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/spider4216/GophProfile/internal/enum"
	"github.com/spider4216/GophProfile/internal/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

type Queue interface {
	SendProcessEvent(ctx context.Context, e models.AvatarProcessEvent) error
}

type S3Client interface {
	InitBucket() error
	Upload(ctx context.Context, key string, reader io.ReadSeeker, ctype string) error
	DeleteAva(ctx context.Context, key string) error
	Download(ctx context.Context, key string) ([]byte, error)
}

type Repository interface {
	GetAvatarByID(ctx context.Context, ID string) (*models.Avatar, error)
	UpdateAvatarUplStatus(ctx context.Context, ID string, status enum.UploadStatus) error
	CommitProcess(ctx context.Context, avatarID string, thumbBytes []byte) error
	DeleteAvatar(ctx context.Context, ID string) error
}

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, trace.Span)
}

type Service struct {
	logger *slog.Logger
	queue  Queue
	repo   Repository
	s3Cli  S3Client
	tracer Tracer
}

func NewService(logger *slog.Logger, q Queue, repo Repository, s3Cli S3Client, tracer Tracer) *Service {
	return &Service{
		logger: logger,
		queue:  q,
		repo:   repo,
		s3Cli:  s3Cli,
		tracer: tracer,
	}
}

func (s *Service) Upload(ctx context.Context, e models.AvatarUploadEvent) error {
	ctx, span := s.tracer.Start(ctx, "UploadAvatarConsumeProcess")
	defer span.End()

	ava, err := s.repo.GetAvatarByID(ctx, e.AvatarID)
	if err != nil {
		return fmt.Errorf("cannot get ava from db: %w", err)
	}

	ext := filepath.Ext(ava.FileName)
	name := strings.TrimSuffix(ava.FileName, ext)

	filename := name + "_" + e.S3Key + ext

	span.SetAttributes(attribute.String("filename", filename))

	file, err := os.Open("/tmp/" + filename)
	if err != nil {
		return fmt.Errorf("cannot open tmp file")
	}

	s.logger.Debug("upload avatar to minio", "avatar", e.AvatarID)

	if err := s.s3Cli.Upload(ctx, e.S3Key, file, ava.MimeType); err != nil {
		return fmt.Errorf("cannot upload to minio: %w", err)
	}

	if err := s.repo.UpdateAvatarUplStatus(ctx, ava.ID, enum.Uploaded); err != nil {
		return fmt.Errorf("cannot update avatar status: %w", err)
	}

	if err := os.Remove("/tmp/" + filename); err != nil {
		return fmt.Errorf("cannot remove tmp file: %w", err)
	}

	return nil
}

func (s *Service) ProcessAvatar(ctx context.Context, e models.AvatarProcessEvent, quality int) error {
	ctx, span := s.tracer.Start(ctx, "ProceeAvatarConsume")
	defer span.End()
	sc := trace.SpanContextFromContext(ctx)

	// Извлечь из БД аватар
	ava, err := s.repo.GetAvatarByID(ctx, e.AvatarID)
	if err != nil {
		return fmt.Errorf("cannot get avatar from db: %w", err)
	}

	// Получить файл из Minio
	b, err := s.s3Cli.Download(ctx, ava.S3Key)
	if err != nil {
		return fmt.Errorf("cannot get avatar from minio: %w", err)
	}

	// Распарсить изображение
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("cannot decode avatar: %w", err)
	}

	thumpnails := make(map[string]string)
	var mu sync.Mutex

	var g errgroup.Group

	// Перебрать все ops
	for _, v := range e.Operations {
		var w, h int

		// Определяем размер из события
		switch v {
		case enum.Size100:
			w, h = 100, 100
		case enum.Size300:
			w, h = 300, 300
		default:
			s.logger.Warn("no any size for process")
			continue
		}

		// Изолируем
		cv := v
		cw, ch := w, h

		// Распаралеливаем обработку и загрузку
		g.Go(func() error {
			s.logger.Debug("process and upload avatar", "id", ava.ID, "size", v, "trace_id", sc.TraceID().String())
			// Для каждого сделать rsize и кроп
			result := imaging.Fill(img, cw, ch, imaging.Center, imaging.Lanczos)

			var buf bytes.Buffer

			if err := jpeg.Encode(&buf, result, &jpeg.Options{
				Quality: quality,
			}); err != nil {
				return fmt.Errorf("cannot encode avatar: %w", err)
			}

			// Каждый кроп загрузить в minio
			key := uuid.NewString()
			if err := s.s3Cli.Upload(ctx, key, bytes.NewReader(buf.Bytes()), ava.MimeType); err != nil {
				return fmt.Errorf("cannot upload to minio: %w", err)
			}

			// Кропы накопить и в конечном итоге сохранить в бд в таблицу avatars
			mu.Lock()
			thumpnails[cv.String()] = key
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("cannot process and upload thumbnails: %w", err)
	}

	// Сохранить thumbnails в БД
	thumbBytes, err := json.Marshal(thumpnails)
	if err != nil {
		return fmt.Errorf("cannot marshal thumbnails: %w", err)
	}

	// Обновляем статус и сохраняем thumbnails в БД в транзакции
	if err := s.repo.CommitProcess(ctx, e.AvatarID, thumbBytes); err != nil {
		return fmt.Errorf("cannot commit changes in process avatar for thumbnails: %w", err)
	}

	return nil
}

func (s *Service) DeleteAvatar(ctx context.Context, e *models.AvatarDeleteEvent) error {
	ctx, span := s.tracer.Start(ctx, "DeleteAvatarProcess")
	defer span.End()
	sc := trace.SpanContextFromContext(ctx)

	// Получаем ava
	ava, err := s.repo.GetAvatarByID(ctx, e.AvatarID)
	if err != nil {
		return fmt.Errorf("cannot get ava for deleting: %w", err)
	}

	var g errgroup.Group

	// Удаляем все thumbnails в minio
	for _, s3key := range ava.ThumbnailS3Keys {
		// Замыкаем
		k := s3key
		// thumbIDs = append(thumbIDs, id)
		g.Go(func() error {
			s.logger.Debug("delete key", "key", k)
			return s.s3Cli.DeleteAva(ctx, k)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("error while deleting thumbnail: %w", err)
	}

	// Удаляем основной avatar в minio
	s.logger.Debug("delete key", "key", ava.S3Key, "trace_id", sc.TraceID().String())
	if err := s.s3Cli.DeleteAva(ctx, ava.S3Key); err != nil {
		return fmt.Errorf("cannot delete avatar from minio: %w", err)
	}

	// Удаляем avatar из DB в Soft режиме
	return s.repo.DeleteAvatar(ctx, ava.ID)
}

func (s *Service) SendProcessEvent(ctx context.Context, avaID string) error {
	ctx, span := s.tracer.Start(ctx, "SendProcessEvent")
	defer span.End()
	span.SetAttributes(
		attribute.String("avatarID", avaID),
	)

	e := models.AvatarProcessEvent{
		AvatarID: avaID,
		Operations: []models.ProcessingOp{
			enum.Size100, enum.Size300,
		},
	}

	s.logger.Debug("send process for avatar", "avatar", avaID)

	return s.queue.SendProcessEvent(ctx, e)
}
