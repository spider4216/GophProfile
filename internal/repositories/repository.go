package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	sql := "SELECT id,user_id,file_name,mime_type,size_bytes,s3_key,COALESCE(thumbnail_s3_keys, '{}'::jsonb),upload_status,processing_status,created_at,updated_at,deleted_at FROM avatars WHERE id=$1 and deleted_at IS NULL"

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

// todo подумать возможно объединить с GetAvatarByID  как то
func (repo *Repository) GetUserAvatarByID(ctx context.Context, userID string, ID string) (*models.Avatar, error) {
	sql := "SELECT id,user_id,file_name,mime_type,size_bytes,s3_key,COALESCE(thumbnail_s3_keys, '{}'::jsonb),upload_status,processing_status,created_at,updated_at,deleted_at FROM avatars WHERE id=$1 and user_id=$2 and deleted_at IS NULL"

	var ava models.Avatar

	var thumbnailS3Keys []byte

	err := repo.con.QueryRowContext(ctx, sql, ID, userID).Scan(
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

func (repo *Repository) UpdateThumbnails(ctx context.Context, ID string, thumbnails []byte) error {
	sql := "UPDATE avatars SET thumbnail_s3_keys=$1 WHERE id=$2"

	_, err := repo.con.ExecContext(ctx, sql, thumbnails, ID)

	if err != nil {
		return err
	}

	return nil
}

// todo можно сделать один sql для этой функции и UpdateAvatarUplStatus
func (repo *Repository) UpdateAvatarProcStatus(ctx context.Context, ID string, status enum.ProcStatus) error {
	sql := "UPDATE avatars SET processing_status=$1 WHERE id=$2"

	_, err := repo.con.ExecContext(ctx, sql, status, ID)

	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) CommitProcess(ctx context.Context, avatarID string, thumbBytes []byte) error {
	tx, err := repo.con.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			repo.logger.Warn("cannot rollback in apply sync", "error", err)
		}
	}()

	if err := repo.UpdateThumbnails(ctx, avatarID, thumbBytes); err != nil {
		return fmt.Errorf("cannot update avatar for thumbnails: %w", err)
	}

	// Изменить статус
	if err := repo.UpdateAvatarProcStatus(ctx, avatarID, enum.ProcDone); err != nil {
		return fmt.Errorf("cannot update status on avatar: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction in commit process error: %w", err)
	}

	return nil
}

func (repo *Repository) GetLatestUserAvatar(ctx context.Context, userID string) (*models.Avatar, error) {
	sql := "SELECT id,user_id,file_name,mime_type,size_bytes,s3_key,COALESCE(thumbnail_s3_keys, '{}'::jsonb),upload_status,processing_status,created_at,updated_at,deleted_at FROM avatars WHERE user_id=$1 and deleted_at IS NULL ORDER BY created_at DESC LIMIT 1"

	var ava models.Avatar

	var thumbnailS3Keys []byte

	err := repo.con.QueryRowContext(ctx, sql, userID).Scan(
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
		return nil, fmt.Errorf("cannot unmarshal avatar: %w", err)
	}

	return &ava, nil
}

func (repo *Repository) DeleteAvatar(ctx context.Context, ID string) error {
	sql := "UPDATE avatars SET deleted_at=$1 WHERE id=$2"

	_, err := repo.con.ExecContext(ctx, sql, time.Now(), ID)

	if err != nil {
		return err
	}

	return nil
}

func (repo *Repository) GetUserAvatars(ctx context.Context, userID string) ([]models.Avatar, error) {
	sql := "SELECT id,user_id,file_name,mime_type,size_bytes,s3_key,COALESCE(thumbnail_s3_keys, '{}'::jsonb),upload_status,processing_status,created_at,updated_at,deleted_at FROM avatars WHERE user_id=$1 and deleted_at IS NULL ORDER BY created_at"

	rows, err := repo.con.QueryContext(ctx, sql, userID)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			repo.logger.Warn("cannot close rows", "error", err)
		}
	}()

	var avatars []models.Avatar
	var thumbnailS3Keys []byte

	for rows.Next() {
		var ava models.Avatar

		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(thumbnailS3Keys, &ava.ThumbnailS3Keys); err != nil {
			return nil, fmt.Errorf("cannot unmarshal avatar: %w", err)
		}

		avatars = append(avatars, ava)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return avatars, nil
}
