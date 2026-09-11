package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

	if err := s.s3Cli.Upload(ctx, e.S3Key, file, ava.MimeType); err != nil {
		return fmt.Errorf("cannot upload to minio: %w", err)
	}

	if err := s.repo.UpdateAvatarUplStatus(ctx, ava.ID, enum.Uploaded); err != nil {
		return fmt.Errorf("cannot update avatar status: %w", err)
	}

	return nil
}
