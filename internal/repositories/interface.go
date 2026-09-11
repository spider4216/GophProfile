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
	UpdateAvatarUplStatus(ctx context.Context, ID string, status enum.UploadStatus) error
}
