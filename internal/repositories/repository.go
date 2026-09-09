package repositories

import (
	"context"
	"database/sql"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
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

func (db *Repository) Ping(ctx context.Context) error {
	return db.con.PingContext(ctx)
}
