package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spider4216/GophProfile/internal/config"
	"github.com/spider4216/GophProfile/internal/logger"
	"github.com/spider4216/GophProfile/internal/minio"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/repositories"
	"github.com/spider4216/GophProfile/internal/tracer"
)

type app struct {
	logger         *slog.Logger
	cfg            *config.Config
	queue          *queue.Queue
	s3Client       *minio.S3Client
	repo           *repositories.Repository
	db             *sql.DB
	ctx            context.Context
	ctxStop        context.CancelFunc
	logShutdown    func()
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
		Step((*app).initDB).
		Step((*app).initRepo).
		Step((*app).initQueue).
		Step((*app).initMinio).
		Build()

	return err
}

func (a *app) initQueue() error {
	q, err := queue.NewQueue(a.cfg.RabbitDSN, a.logger, a.tracer)
	if err != nil {
		return fmt.Errorf("cannot init queue: %w", err)
	}

	// Декларируем очереди
	if err = q.DeclareQueues(); err != nil {
		return fmt.Errorf("cannot declare queue: %w", err)
	}

	// Декларируем Exchange
	if err := q.DeclareExchange(); err != nil {
		return fmt.Errorf("cannot declare exchange: %w", err)
	}

	// Делаем Binds Exchange с Queues
	if err := q.QueuesBind(); err != nil {
		return fmt.Errorf("cannot bind queues: %w", err)
	}

	// Декларируем консьюмеры
	if err := q.DeclareConsumers(); err != nil {
		return fmt.Errorf("cannot declare consumers: %w", err)
	}

	a.queue = q

	return nil
}

func (a *app) initConfig() error {
	var cfg *config.Config

	cfg, err := config.New()
	if err != nil {
		return err
	}

	a.cfg = cfg

	return nil
}

func (a *app) initLogger() error {
	logger, shutdown, err := logger.Init(a.ctx, a.cfg.ServiceName, a.cfg.ServiceVersion)
	if err != nil {
		return fmt.Errorf("cannot init logger: %w", err)
	}

	a.logger = logger
	a.logShutdown = shutdown

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

func (a *app) initRepo() error {
	repo := repositories.NewRepository(a.db, a.logger, a.tracer)

	a.repo = repo

	return nil
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	a.ctx = ctx
	a.ctxStop = stop

	return nil
}

func (a *app) initTracer() error {
	t := tracer.NewTracer()
	f, err := t.Init(a.ctx, a.cfg.ServiceName)
	if err != nil {
		return fmt.Errorf("cannot init tracer: %w", err)
	}

	a.tracer = t
	a.tracerShutdown = f

	return nil
}
