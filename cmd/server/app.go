package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	buldCfg "github.com/spider4216/GophProfile/internal/config"
	"github.com/spider4216/GophProfile/internal/logger"
	"github.com/spider4216/GophProfile/internal/repositories"
	"github.com/spider4216/GophProfile/internal/server/config"
	"github.com/spider4216/GophProfile/migrations"
)

type app struct {
	logger *slog.Logger
	cfg    *config.Config
	repo   repositories.RepositoryInterface
}

func newApp() *app {
	return &app{}
}

func (a *app) Run() error {
	_, err := buldCfg.NewBuilder(a).
		Step((*app).initConfig).
		Step((*app).initLogger).
		Step((*app).initRepo).
		Step((*app).initMigrations).
		Build()

	return err
}

func (a *app) initConfig() error {
	var cfg *config.Config

	cfg, err := config.New()
	if err != nil {
		return err
	}

	if cfg.DbDsn == "" {
		return errors.New("dsn didtn't passed")
	}

	a.cfg = cfg

	return nil
}

func (a *app) initLogger() error {
	logger := logger.Init(a.cfg.LogLvl)

	a.logger = logger

	return nil
}

func (a *app) initRepo() error {
	repo, err := repositories.NewRepository(a.cfg.DbDsn, a.logger)
	if err != nil {
		return err
	}

	a.repo = repo

	return nil
}

func (a *app) initMigrations() error {
	a.logger.Debug("Up migrations")

	repo, ok := a.repo.(*repositories.Repository)

	if !ok {
		return fmt.Errorf("cannot cast to pgx repository type in init migration")
	}

	src, ok := repo.Source().(*sql.DB)

	if !ok {
		return fmt.Errorf("cannot cast to sql.DB type in init migration")
	}

	if err := migrations.Run(src); err != nil {
		return err
	}

	a.logger.Debug("Migration done")

	return nil
}
