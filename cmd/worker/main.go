package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rabbitmq/amqp091-go"
	"github.com/spider4216/GophProfile/internal/worker/handlers"
	"github.com/spider4216/GophProfile/internal/worker/services"
)

func main() {
	app := newApp()

	if err := app.Run(); err != nil {
		log.Fatal("Cannot run app", err)
	}

	service := services.NewService(app.logger, app.queue, app.repo, app.s3Client)
	handler := handlers.NewHandler(app.logger, service, app.cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.logger.Debug("Run consumers...")

	app.runConsume(ctx, handler)

	if err := app.shutdown(); err != nil {
		app.logger.Warn("cannot shutdown correctly")
	}
}

func (app *app) runConsume(ctx context.Context, handler *handlers.Handler) {
	for {
		select {
		case d := <-app.queue.GetUploadConsumer():
			app.logger.Debug("Consume upload...")
			consume(ctx, d, handler.UploadAvatar, app.logger)
		case d := <-app.queue.GetDeleteConsumer():
			app.logger.Debug("Consume delete...")
			consume(ctx, d, handler.DeleteAvatar, app.logger)
		case d := <-app.queue.GetProcessConsumer():
			app.logger.Debug("Consume process...")
			consume(ctx, d, handler.ProcessAvatar, app.logger)
		case <-ctx.Done():
			return
		}
	}
}

func (app *app) shutdown() error {
	if err := app.queue.ConnectClose(); err != nil {
		return fmt.Errorf("cannot close queue connection")
	}

	return nil
}

func consume[T any](
	ctx context.Context,
	delivery amqp091.Delivery,
	f func(ctx context.Context, e T) error,
	logger *slog.Logger,
) {
	var event T

	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		logger.Error("Cannot unmarshall", "error", err)
		if err := delivery.Nack(false, false); err != nil {
			logger.Warn("cannot nack in consume")
		}

		return
	}

	if err := f(ctx, event); err != nil {
		logger.Error("cannot consume", "error", err)

		if err := delivery.Nack(false, false); err != nil {
			logger.Warn("cannot nack in consume")
		}
		return
	}

	if err := delivery.Ack(false); err != nil {
		logger.Warn("cannot ack in consume")
	}
}
