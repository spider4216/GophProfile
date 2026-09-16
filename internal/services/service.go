package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spider4216/GophProfile/internal/enum"
	"github.com/spider4216/GophProfile/internal/minio"
	"github.com/spider4216/GophProfile/internal/models"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/repositories"
	srvModel "github.com/spider4216/GophProfile/internal/server/models"
)

type Service struct {
	repo   repositories.RepositoryInterface
	logger *slog.Logger
	queue  queue.QueueInterface
	s3Cli  minio.S3ClientInterface
}

func New(repo repositories.RepositoryInterface, logger *slog.Logger, queue queue.QueueInterface, s3Cli minio.S3ClientInterface) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		queue:  queue,
		s3Cli:  s3Cli,
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

func (s *Service) SendDeleteEvent(ctx context.Context, avaID string) error {
	e := models.AvatarDeleteEvent{
		AvatarID: avaID,
	}

	return s.queue.SendDeleteEvent(ctx, e)
}

func (s *Service) SendDeleteEvents(ctx context.Context, avas []models.Avatar) error {
	for _, ava := range avas {
		s.logger.Debug("Send to delete ava", "id", ava.ID)
		e := models.AvatarDeleteEvent{
			AvatarID: ava.ID,
		}

		if err := s.queue.SendDeleteEvent(ctx, e); err != nil {
			return fmt.Errorf("cannit send event to delete avatar: %w", err)
		}
	}

	return nil
}

func (s *Service) GetComplexBinaryAva(ctx context.Context, size string, ava *models.Avatar) ([]byte, int, error) {
	if size == "" {
		if ava.UploadStatus != enum.Uploaded.String() {
			s.logger.Error("avatar uploading... try again latter", "avaid", ava.ID)
			return nil, http.StatusServiceUnavailable, errors.New("avatar uploading... try again latter")
		}

		b, err := s.GetBinaryAva(ctx, ava.S3Key)
		if err != nil {
			s.logger.Error("cannot download original avatar", "error", err)
			return nil, http.StatusNotFound, fmt.Errorf("cannot download original avatar: %w", err)
		}

		return b, 0, nil
	}

	if ava.ProcessingStatus != enum.ProcDone.String() {
		s.logger.Error("avatar thumbnails processing... try again latter", "avaid", ava.ID)

		return nil, http.StatusServiceUnavailable, errors.New("avatar thumbnails processing... try again latter")
	}

	b, err := s.GetBinaryThumbnail(ctx, ava, size)
	if err != nil {
		s.logger.Error("cannot download thumbnail avatar", "error", err)

		return nil, http.StatusNotFound, fmt.Errorf("cannot download thumbnail avatar: %w", err)
	}

	return b, 0, nil
}

func (s *Service) GetBinaryAva(ctx context.Context, s3key string) ([]byte, error) {
	return s.s3Cli.Download(ctx, s3key)
}

func (s *Service) GetBinaryThumbnail(ctx context.Context, ava *models.Avatar, size string) ([]byte, error) {
	s3Key, ok := ava.ThumbnailS3Keys[size]

	if !ok {
		return nil, fmt.Errorf("cannot fine s3key thumbnail by size: %s", size)
	}

	return s.s3Cli.Download(ctx, s3Key)
}

func (s *Service) GetAvatarByID(ctx context.Context, ID string) (*models.Avatar, error) {
	return s.repo.GetAvatarByID(ctx, ID)
}

func (s *Service) HashBinary(data []byte) string {
	hash := sha256.Sum256(data)

	return fmt.Sprintf("%x", hash)
}

func (s *Service) PrepareNotFoundResp() ([]byte, error) {
	resp := srvModel.GetNoAvaResp{
		Err: "Avatar not found",
	}

	return json.Marshal(resp)
}

func (s *Service) PrepareForbiddenResp() ([]byte, error) {
	resp := srvModel.ForbiddenResp{
		Error:   "Forbidden",
		Details: "You can only delete your own avatars",
	}

	return json.Marshal(resp)
}

func (s *Service) GetLatestActiveUserAvatar(ctx context.Context, userID string) (*models.Avatar, error) {
	return s.repo.GetLatestUserAvatar(ctx, userID)
}

func (s *Service) GetUserAvatars(ctx context.Context, userID string) ([]models.Avatar, error) {
	return s.repo.GetUserAvatars(ctx, userID)
}
