package reptest

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/spider4216/GophProfile/internal/enum"
	"github.com/spider4216/GophProfile/internal/models"
)

type SliceRepository struct {
	data   map[string][][]byte
	logger *slog.Logger
}

func NewRepository(logger *slog.Logger, store map[string][][]byte) *SliceRepository {
	return &SliceRepository{
		data:   store,
		logger: logger,
	}
}

func (r *SliceRepository) Ping(ctx context.Context) error {
	return nil
}

func (r *SliceRepository) CreateAvatar(ctx context.Context, ava models.Avatar) (string, error) {
	ava.ID = uuid.NewString()

	b, err := json.Marshal(ava)
	if err != nil {
		return "", err
	}

	r.data["avatars"] = append(r.data["avatars"], b)

	return ava.ID, nil
}

func (r *SliceRepository) GetAvatarByID(ctx context.Context, ID string) (*models.Avatar, error) {
	return nil, nil
}

func (r *SliceRepository) UpdateAvatarUplStatus(ctx context.Context, ID string, status enum.UploadStatus) error {
	return nil
}

func (r *SliceRepository) UpdateAvatarProcStatusTx(ctx context.Context, tx *sql.Tx, ID string, status enum.ProcStatus) error {
	return nil
}

func (r *SliceRepository) UpdateThumbnailsTx(ctx context.Context, tx *sql.Tx, ID string, thumbnails []byte) error {
	return nil
}

func (r *SliceRepository) CommitProcess(ctx context.Context, avatarID string, thumbBytes []byte) error {
	return nil
}

func (r *SliceRepository) GetLatestUserAvatar(ctx context.Context, userID string) (*models.Avatar, error) {
	return nil, nil
}

func (r *SliceRepository) DeleteAvatar(ctx context.Context, ID string) error {
	return nil
}

func (r *SliceRepository) GetUserAvatars(ctx context.Context, userID string) ([]models.Avatar, error) {
	return nil, nil
}
