package repositories

import (
	"context"

	"github.com/spider4216/GophProfile/internal/enum"
	"github.com/spider4216/GophProfile/internal/models"
)

type RepositoryInterface interface {
	Ping(ctx context.Context) error
	Source() any
	CreateAvatar(ctx context.Context, ava models.Avatar) (string, error)
	GetAvatarByID(ctx context.Context, ID string) (*models.Avatar, error)
	GetUserAvatarByID(ctx context.Context, userID string, ID string) (*models.Avatar, error)
	UpdateAvatarUplStatus(ctx context.Context, ID string, status enum.UploadStatus) error
	UpdateAvatarProcStatus(ctx context.Context, ID string, status enum.ProcStatus) error
	UpdateThumbnails(ctx context.Context, ID string, thumbnails []byte) error
	CommitProcess(ctx context.Context, avatarID string, thumbBytes []byte) error
	GetLatestUserAvatar(ctx context.Context, userID string) (*models.Avatar, error)
	DeleteAvatar(ctx context.Context, ID string) error
	GetUserAvatars(ctx context.Context, userID string) ([]models.Avatar, error)
}
