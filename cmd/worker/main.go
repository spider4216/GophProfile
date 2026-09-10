package main

import (
	"context"
	"log"
	"os"

	"github.com/spider4216/GophProfile/internal/worker/handlers"
	"github.com/spider4216/GophProfile/internal/worker/services"
)

func main() {
	app := newApp()

	if err := app.Run(); err != nil {
		log.Fatal("Cannot run app", err)
	}

	service := services.NewService(app.logger, app.queue)
	handler := handlers.NewHandler(app.logger, service)

	// todo ctx with timeout
	ctx := context.Background()

	app.logger.Debug("Run consumers...")

	// todo gracefull shutdown

	var err error

	for {
		select {
		case d := <-app.queue.UploadConsumer:
			app.logger.Debug("Consume upload...")

			err = handler.UploadAvatar(ctx)

			if err == nil {
				d.Ack(false)
			}
		case d := <-app.queue.DeleteConsumer:
			app.logger.Debug("Consume delete...")
			d.Ack(false)
		case d := <-app.queue.ProcessConsumer:
			app.logger.Debug("Consume process...")
			d.Ack(false)
		}

		if err != nil {
			app.logger.Error("something went wrong", "error", err)
			os.Exit(1)
		}
	}

}
