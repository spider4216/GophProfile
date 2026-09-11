package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/spider4216/GophProfile/internal/enum"
	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/repositories"
	"github.com/spider4216/GophProfile/internal/worker/minio"
)

type Service struct {
	logger *slog.Logger
	queue  *queue.Queue
	repo   repositories.RepositoryInterface
	s3Cli  *minio.S3Client
}

func NewService(logger *slog.Logger, q *queue.Queue, repo repositories.RepositoryInterface, s3Cli *minio.S3Client) *Service {
	return &Service{
		logger: logger,
		queue:  q,
		repo:   repo,
		s3Cli:  s3Cli,
	}
}

func (s *Service) Upload(ctx context.Context, e models.AvatarUploadEvent) error {
	ava, err := s.repo.GetAvatarByID(ctx, e.AvatarID)

	if err != nil {
		return fmt.Errorf("cannot get ava from db: %w", err)
	}

	ext := filepath.Ext(ava.FileName)
	name := strings.TrimSuffix(ava.FileName, ext)

	filename := name + "_" + e.S3Key + ext

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
		s.logger.Warn("cannot delete file from tmp", "error", err)
	}

	return nil
}

func (s *Service) ProcessAvatar(ctx context.Context, e models.AvatarProcessEvent, quality int) error {
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

	// Перебрать все ops
	// todo errgroup
	for _, v := range e.Operations {
		var w, h int

		if v == enum.Size100 {
			w, h = 100, 100
		}

		if v == enum.Size300 {
			w, h = 300, 300
		}

		// Для каждого сделать кроп
		result := imaging.Fill(
			img,
			w,
			h,
			imaging.Center,
			imaging.Lanczos,
		)

		var buf bytes.Buffer

		if err := jpeg.Encode(&buf, result, &jpeg.Options{
			Quality: quality,
		}); err != nil {
			return fmt.Errorf("cannot encode avatar: %w", err)
		}

		// Каждый кроп загрузить в minio
		key := uuid.NewString()
		// todo преобразовать во все форматы jpg, webp, png, пока буду использовать один формат
		if err := s.s3Cli.Upload(ctx, key, bytes.NewReader(buf.Bytes()), ava.MimeType); err != nil {
			return fmt.Errorf("cannot upload to minio: %w", err)
		}

		// Кропы накопить и в конечном итоге сохранить в бд в таблицу avatars
		thumpnails[v.String()] = key
	}

	// todo транзакция

	// Сохранить thumbnails в БД
	thumbBytes, err := json.Marshal(thumpnails)

	if err != nil {
		return fmt.Errorf("cannot marshal thumbnails: %w", err)
	}

	if err := s.repo.UpdateThumbnails(ctx, e.AvatarID, thumbBytes); err != nil {
		return fmt.Errorf("cannot update avatar for thumbnails: %w", err)
	}

	// Изменить статус
	if err := s.repo.UpdateAvatarUplStatus(ctx, e.AvatarID, enum.Uploaded); err != nil {
		return fmt.Errorf("cannot update status on avatar: %w", err)
	}

	return nil
}

func (s *Service) SendProcessEvent(ctx context.Context, avaID string) error {
	e := models.AvatarProcessEvent{
		AvatarID: avaID,
		Operations: []models.ProcessingOp{
			enum.Size100, enum.Size300,
		},
	}

	s.logger.Debug("send process for avatar", "avatar", avaID)

	return s.queue.SendProcessEvent(ctx, e)
}
