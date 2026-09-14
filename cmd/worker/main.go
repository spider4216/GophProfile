package main

import (
	"context"
	"encoding/json"
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

	for {
		select {
		case d := <-app.queue.UploadConsumer:
			app.logger.Debug("Consume upload...")
			consume(ctx, d, handler.UploadAvatar, app.logger)
		case d := <-app.queue.DeleteConsumer:
			app.logger.Debug("Consume delete...")
			consume(ctx, d, handler.DeleteAvatar, app.logger)
		case d := <-app.queue.ProcessConsumer:
			app.logger.Debug("Consume process...")
			consume(ctx, d, handler.ProcessAvatar, app.logger)
		case <-ctx.Done():
			os.Exit(1)
		}
	}
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
		return
	}

	if err := f(ctx, event); err != nil {
		logger.Error("cannot consume", "error", err)
		return
	}

	if err := delivery.Ack(false); err != nil {
		logger.Warn("cannot ack in consume delete")
	}
}
