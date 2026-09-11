package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
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
