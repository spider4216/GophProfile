package main

import (
	"fmt"
	"log/slog"

	buldCfg "github.com/spider4216/GophProfile/internal/config"
	"github.com/spider4216/GophProfile/internal/logger"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/worker/config"
)

type app struct {
	logger *slog.Logger
	cfg    *config.Config
	queue  *queue.Queue
}

func newApp() *app {
	return &app{}
}

func (a *app) Run() error {
	_, err := buldCfg.NewBuilder(a).
		Step((*app).initConfig).
		Step((*app).initLogger).
		Step((*app).initQueue).
		Build()

	return err
}

func (a *app) initQueue() error {
	q, err := queue.NewQueue(a.cfg.RabbitDSN, a.logger)

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
	logger := logger.Init(a.cfg.LogLvl)

	a.logger = logger

	return nil
}
