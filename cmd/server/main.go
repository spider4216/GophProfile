package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spider4216/GophProfile/internal/server/handlers"
	"github.com/spider4216/GophProfile/internal/server/middlewares"
	"github.com/spider4216/GophProfile/internal/services"
)

const (
	serverTimeout time.Duration = 5 * time.Second
)

func main() {
	app := newApp()

	if err := app.Run(); err != nil {
		log.Fatal("Cannot run app", err)
	}

	service := services.New(app.repo, app.logger, app.queue)
	middleware := middlewares.New(app.logger, app.cfg)
	handler := handlers.New(app.cfg, app.logger, service)

	mux := http.NewServeMux()

	mux.Handle("GET /health", middleware.WithLogging(http.HandlerFunc(handler.Health)))
	mux.Handle("POST /api/v1/avatars", middleware.WithLogging(http.HandlerFunc(handler.UploadAvatar)))

	srv := &http.Server{
		Addr:         app.cfg.ServerAddress,
		Handler:      mux,
		ReadTimeout:  app.cfg.ReadTimeout,
		WriteTimeout: app.cfg.WriteTimeout,
		IdleTimeout:  app.cfg.IdleTimeout,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	go func() {
		defer wg.Done()

		app.logger.Debug("Graceful shutdown mode on")
		<-ctx.Done()

		// Тут тоже останавляваем перехват сигналов
		stop()

		app.logger.Debug("Shutdown server...")

		ctxShutdown, cancel := context.WithTimeout(context.Background(), serverTimeout)
		defer cancel()

		if err := srv.Shutdown(ctxShutdown); err != nil {
			app.logger.Warn("Cannot shutdown main server", "error", err)
		}
	}()

	app.logger.Debug("Listen server", "address", app.cfg.ServerAddress)

	if err := srv.ListenAndServeTLS(app.cfg.CrtPath, app.cfg.PKPath); err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.logger.Error("Server error", "error", err)
		os.Exit(1)
	}

	wg.Wait()
}
