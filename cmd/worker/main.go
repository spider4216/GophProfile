package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/spider4216/GophProfile/internal/queue"
	"github.com/spider4216/GophProfile/internal/tracer"
	"github.com/spider4216/GophProfile/internal/worker/handlers"
	"github.com/spider4216/GophProfile/internal/worker/services"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	app := newApp()

	if err := app.Init(); err != nil {
		log.Fatal("Cannot run app", err)
	}

	service := services.NewService(app.logger, app.queue, app.repo, app.s3Client, app.tracer, app.meter)
	handler := handlers.NewHandler(app.logger, service, app.cfg, app.tracer, app.meter)
	srvHandler := handlers.NewSrvHandler(app.queue, app.logger)

	mux := http.NewServeMux()

	mux.Handle("GET /liveness", http.HandlerFunc(srvHandler.Liveness))
	mux.Handle("GET /readiness", http.HandlerFunc(srvHandler.Readiness))

	srv := &http.Server{
		Addr:         app.cfg.ServerAddress,
		Handler:      mux,
		ReadTimeout:  app.cfg.ReadTimeout,
		WriteTimeout: app.cfg.WriteTimeout,
		IdleTimeout:  app.cfg.IdleTimeout,
	}

	go func() {
		app.logger.Info("Run Liveness & Readiness probs in worker")

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("Server error", "error", err)
		}
	}()

	defer app.ctxStop()

	app.logger.Debug("Run consumers...")

	app.runConsume(app.ctx, handler)

	if err := app.shutdown(srv); err != nil {
		app.logger.Warn("cannot shutdown correctly", "error", err)
	}
}

func (app *app) runConsume(ctx context.Context, handler *handlers.Handler) {
	for {
		select {
		case d := <-app.queue.GetUploadConsumer():
			app.logger.Debug("Consume upload...")
			consume(ctx, d, handler.UploadAvatar, app.logger, app.tracer)
		case d := <-app.queue.GetDeleteConsumer():
			app.logger.Debug("Consume delete...")
			consume(ctx, d, handler.DeleteAvatar, app.logger, app.tracer)
		case d := <-app.queue.GetProcessConsumer():
			app.logger.Debug("Consume process...")
			consume(ctx, d, handler.ProcessAvatar, app.logger, app.tracer)
		case <-ctx.Done():
			return
		}
	}
}

func (app *app) shutdown(srv *http.Server) error {
	// Контекст для завершения сервера
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("cannot shutdown server: %w", err)
	}

	app.logShutdown()
	app.tracerShutdown()
	app.meterShutdown()

	if err := app.queue.ConnectClose(); err != nil {
		return fmt.Errorf("cannot close queue connection: %w", err)
	}

	return nil
}

func consume[T any](
	ctx context.Context,
	delivery amqp091.Delivery,
	f func(ctx context.Context, e T) error,
	logger *slog.Logger,
	tracer *tracer.Tracer,
) {
	ctx = otel.GetTextMapPropagator().Extract(
		ctx,
		queue.AMQPCarrier(delivery.Headers),
	)

	ctx, span := tracer.Start(ctx, "ConsumeEvent")
	defer span.End()
	sc := trace.SpanContextFromContext(ctx)

	logger.Debug("Consume", "trace_id", sc.TraceID().String())

	var event T

	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		logger.Error("Cannot unmarshall", "error", err, "trace_id", sc.TraceID().String())
		if err := delivery.Nack(false, false); err != nil {
			logger.Warn("cannot nack in consume", "error", err, "trace_id", sc.TraceID().String())
		}

		return
	}

	if err := f(ctx, event); err != nil {
		logger.Error("cannot consume", "error", err, "trace_id", sc.TraceID().String())

		if err := delivery.Nack(false, false); err != nil {
			logger.Warn("cannot nack in consume", "error", err, "trace_id", sc.TraceID().String())
		}
		return
	}

	if err := delivery.Ack(false); err != nil {
		logger.Warn("cannot ack in consume", "error", err, "trace_id", sc.TraceID().String())
	}
}
