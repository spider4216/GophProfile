package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/repositories"
)

type Service struct {
	repo   repositories.RepositoryInterface
	logger *slog.Logger
	queue  *queue.Queue
}

func New(repo repositories.RepositoryInterface, logger *slog.Logger, queue *queue.Queue) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		queue:  queue,
	}
}

func (s *Service) IsDBOK(ctx context.Context) bool {
	return s.repo.Ping(ctx) == nil
}

func (s *Service) CreateAvatar(ctx context.Context, fname string, mtype string, size int64, s3Key string) (*models.Avatar, error) {
	ava := models.Avatar{
		UserID:    s.GetUserIdFromCtx(ctx),
		FileName:  fname,
		MimeType:  mtype,
		SizeBytes: size,
		S3Key:     s3Key,
	}

	id, err := s.repo.CreateAvatar(ctx, ava)

	if err != nil {
		return nil, fmt.Errorf("cannot create avatar: %w", err)
	}

	ava.ID = id

	return &ava, nil
}

func (s *Service) CreateTmpFile(ctx context.Context, filename string, file io.Reader, uid string) error {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	filename = name + "_" + uid + ext

	tmp, err := os.Create("/tmp/" + filename)

	if err != nil {
		return fmt.Errorf("cannot create tmp file: %w", err)
	}

	defer func() {
		if err := tmp.Close(); err != nil {
			s.logger.Warn("cannot close tmp file", "error", err)
		}
	}()

	_, err = io.Copy(tmp, file)

	if err != nil {
		return fmt.Errorf("cannot put file into tmp: %w", err)
	}

	return nil
}

func (s *Service) SendUploadEvent(ctx context.Context, userID string, avaID string, s3k string) error {
	e := models.AvatarUploadEvent{
		AvatarID: avaID,
		UserID:   userID,
		S3Key:    s3k,
	}

	return s.queue.SendUploadEvent(ctx, e)
}
