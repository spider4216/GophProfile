package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/spider4216/GophProfile/internal/server/handlers"
	"github.com/spider4216/GophProfile/internal/server/middlewares"
	"github.com/spider4216/GophProfile/internal/services"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	serverTimeout time.Duration = 5 * time.Second
)

func main() {
	app := newApp()

	if err := app.Run(); err != nil {
		log.Fatal("Cannot run app", err)
	}

	service := services.New(app.repo, app.logger, app.queue, app.s3Client, app.meter)
	middleware := middlewares.New(app.logger, app.cfg, service)
	handler := handlers.New(app.cfg, app.logger, service, app.tracer)

	mux := http.NewServeMux()

	mux.Handle("GET /", otelhttp.NewHandler(http.FileServer(http.Dir("./web")), "web"))
	mux.Handle("GET /health", otelhttp.NewHandler(middleware.WithLogging(http.HandlerFunc(handler.Health)), "health"))
	mux.Handle("POST /api/v1/avatars", otelhttp.NewHandler(middleware.WithLogging(middleware.WithUser(http.HandlerFunc(handler.UploadAvatar))), "upload_avatar"))
	mux.Handle("GET /api/v1/avatars/{avatar_id}", otelhttp.NewHandler(middleware.WithLogging(http.HandlerFunc(handler.GetAvatar)), "get_avatar"))
	mux.Handle("GET /api/v1/users/{user_id}/avatar", otelhttp.NewHandler(middleware.WithLogging(http.HandlerFunc(handler.GetUserAvatar)), "get_user_avatar"))
	mux.Handle("GET /api/v1/avatars/{avatar_id}/metadata", otelhttp.NewHandler(middleware.WithLogging(http.HandlerFunc(handler.GetMetaAvatar)), "get_meta_avatar"))
	mux.Handle("DELETE /api/v1/avatars/{id}", otelhttp.NewHandler(middleware.WithLogging(middleware.WithUser(http.HandlerFunc(handler.DeleteAvatar))), "delete_avatar"))
	mux.Handle("DELETE /api/v1/users/{user_id}/avatar", otelhttp.NewHandler(middleware.WithLogging(middleware.WithUser(http.HandlerFunc(handler.DeleteUserAvatars))), "delete_user_avatar"))
	mux.Handle("GET /api/v1/users/{user_id}/avatars", otelhttp.NewHandler(middleware.WithLogging(http.HandlerFunc(handler.GetUserAvatars)), "get_user_avatars"))

	srv := &http.Server{
		Addr:         app.cfg.ServerAddress,
		Handler:      mux,
		ReadTimeout:  app.cfg.ReadTimeout,
		WriteTimeout: app.cfg.WriteTimeout,
		IdleTimeout:  app.cfg.IdleTimeout,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	defer app.ctxStop()
	defer app.logShutdown()

	go func() {
		defer wg.Done()

		app.logger.Debug("Graceful shutdown mode on")
		<-app.ctx.Done()

		// Тут тоже останавляваем перехват сигналов
		app.ctxStop()

		app.logger.Debug("Shutdown server...")

		app.logShutdown()

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
