package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spider4216/GophProfile/internal/enum"
	"github.com/spider4216/GophProfile/internal/models"
)

type Repository struct {
	con    *sql.DB
	logger *slog.Logger
}

func NewRepository(dsn string, logger *slog.Logger) (RepositoryInterface, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return &Repository{con: db, logger: logger}, nil
}

func (repo *Repository) Source() any {
	return repo.con
}

func (repo *Repository) Ping(ctx context.Context) error {
	return repo.con.PingContext(ctx)
}

func (repo *Repository) CreateAvatar(ctx context.Context, ava models.Avatar) (string, error) {
	sql := "INSERT INTO avatars (user_id,file_name,mime_type,size_bytes,s3_key) VALUES ($1,$2,$3,$4,$5) RETURNING id"
	var lastInsertId string

	err := repo.con.QueryRowContext(ctx, sql, ava.UserID, ava.FileName, ava.MimeType, ava.SizeBytes, ava.S3Key).Scan(&lastInsertId)

	if err != nil {
		return "", fmt.Errorf("cannot insert ava: %w", err)
	}

	return lastInsertId, nil
}

func (repo *Repository) GetAvatarByID(ctx context.Context, ID string) (*models.Avatar, error) {
	sql := "SELECT id,user_id,file_name,mime_type,size_bytes,s3_key,COALESCE(thumbnail_s3_keys, '{}'::jsonb),upload_status,processing_status,created_at,updated_at,deleted_at FROM avatars WHERE id=$1"

	var ava models.Avatar

	var thumbnailS3Keys []byte

	err := repo.con.QueryRowContext(ctx, sql, ID).Scan(
		&ava.ID,
		&ava.UserID,
		&ava.FileName,
		&ava.MimeType,
		&ava.SizeBytes,
		&ava.S3Key,
		&thumbnailS3Keys,
		&ava.UploadStatus,
		&ava.ProcessingStatus,
		&ava.CreatedAt,
		&ava.UpdatedAt,
		&ava.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("cannot scan: %w", err)
	}

	if err := json.Unmarshal(thumbnailS3Keys, &ava.ThumbnailS3Keys); err != nil {
		return nil, fmt.Errorf("cannot unmarshal thumbnails: %w", err)
	}

	return &ava, nil
}

func (repo *Repository) UpdateAvatarUplStatus(ctx context.Context, ID string, status enum.UploadStatus) error {
	sql := "UPDATE avatars SET upload_status=$1 WHERE id=$2"

	_, err := repo.con.ExecContext(ctx, sql, status, ID)

	if err != nil {
		return err
	}

	return nil
}
