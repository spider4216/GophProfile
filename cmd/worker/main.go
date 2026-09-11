package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/spider4216/GophProfile/internal/models"
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

	// todo ctx with timeout
	ctx := context.Background()

	app.logger.Debug("Run consumers...")

	// todo gracefull shutdown

	for {
		select {
		case d := <-app.queue.UploadConsumer:
			app.logger.Debug("Consume upload...")

			var event models.AvatarUploadEvent

			if err := json.Unmarshal(d.Body, &event); err != nil {
				app.logger.Error("Cannot unmarshall", "error", err)
				break
			}

			if err := handler.UploadAvatar(ctx, event); err != nil {
				app.logger.Error("Cannot upload avatar", "error", err)
				break
			}

			d.Ack(false)
		case d := <-app.queue.DeleteConsumer:
			app.logger.Debug("Consume delete...")
			d.Ack(false)
		case d := <-app.queue.ProcessConsumer:
			app.logger.Debug("Consume process...")

			var event models.AvatarProcessEvent

			if err := json.Unmarshal(d.Body, &event); err != nil {
				app.logger.Error("Cannot unmarshall", "error", err)
				break
			}

			if err := handler.ProcessAvatar(ctx, event); err != nil {
				app.logger.Error("Cannot process avatar", "error", err)
				break
			}

			d.Ack(false)
		}
	}
}
