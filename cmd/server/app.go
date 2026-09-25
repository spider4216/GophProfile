package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/spider4216/GophProfile/internal/config"
	"github.com/spider4216/GophProfile/internal/logger"
	"github.com/spider4216/GophProfile/internal/meter"
	"github.com/spider4216/GophProfile/internal/minio"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/repositories"
	"github.com/spider4216/GophProfile/internal/tracer"
	"github.com/spider4216/GophProfile/migrations"
)

type app struct {
	logger         *slog.Logger
	cfg            *config.Config
	repo           *repositories.Repository
	s3Client       *minio.S3Client
	queue          *queue.Queue
	db             *sql.DB
	ctx            context.Context
	ctxStop        context.CancelFunc
	logShutdown    func()
	meter          *meter.Meter
	meterShutdown  func()
	tracer         *tracer.Tracer
	tracerShutdown func()
}

func newApp() *app {
	return &app{}
}

func (a *app) Run() error {
	_, err := config.NewBuilder(a).
		Step((*app).initCtx).
		Step((*app).initConfig).
		Step((*app).initTracer).
		Step((*app).initLogger).
		Step((*app).initMeter).
		Step((*app).initDB).
		Step((*app).initRepo).
		Step((*app).initMigrations).
		Step((*app).initQueue).
		Step((*app).initMinio).
		Build()

	return err
}

func (a *app) initConfig() error {
	var cfg *config.Config

	cfg, err := config.New()
	if err != nil {
		return err
	}

	if cfg.DbDSN == "" {
		return errors.New("dsn didtn't passed")
	}

	a.cfg = cfg

	return nil
}

func (a *app) initLogger() error {
	logger, shutdown, err := logger.Init(a.ctx)
	if err != nil {
		return fmt.Errorf("cannot init logger: %w", err)
	}

	a.logger = logger
	a.logShutdown = shutdown

	return nil
}

func (a *app) initRepo() error {
	repo := repositories.NewRepository(a.db, a.logger)

	a.repo = repo

	return nil
}

func (a *app) initMigrations() error {
	a.logger.Debug("Up migrations")

	if err := migrations.Run(a.db); err != nil {
		return err
	}

	a.logger.Debug("Migration done")

	return nil
}

func (a *app) initQueue() error {
	q, err := queue.NewQueue(a.cfg.RabbitDSN, a.logger)
	if err != nil {
		return fmt.Errorf("cannot init queue: %w", err)
	}

	// Декларируем очереди
	err = q.DeclareQueues()
	// Декларируем routing keys и binds делаются на
	// стороне workers
	if err != nil {
		return fmt.Errorf("cannot declare queue: %w", err)
	}

	a.queue = q

	return nil
}

func (a *app) initMinio() error {
	cli, err := minio.NewS3Client(a.cfg.MinioUser, a.cfg.MinioPass, a.cfg.MinioHost, a.cfg.BucketName, a.logger)
	if err != nil {
		return fmt.Errorf("cannot create s3 client: %w", err)
	}

	a.s3Client = cli

	return a.s3Client.InitBucket()
}

func (a *app) initDB() error {
	db, err := sql.Open("pgx", a.cfg.DbDSN)
	if err != nil {
		return err
	}

	a.db = db

	return nil
}

func (a *app) initCtx() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	a.ctx = ctx
	a.ctxStop = stop

	return nil
}

func (a *app) initMeter() error {
	m := meter.NewMeter()
	f, err := m.Init(a.ctx)

	if err != nil {
		return fmt.Errorf("cannot init meter: %w", err)
	}

	a.meter = m
	a.meterShutdown = f

	return nil
}

func (a *app) initTracer() error {
	t := tracer.NewTracer()
	f, err := t.Init(a.ctx)

	if err != nil {
		return fmt.Errorf("cannot init tracer: %w", err)
	}

	a.tracer = t
	a.tracerShutdown = f

	return nil
}
