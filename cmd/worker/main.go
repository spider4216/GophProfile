package main

import "log"

func main() {
	app := newApp()

	if err := app.Run(); err != nil {
		log.Fatal("Cannot run app", err)
	}

	app.logger.Debug("Run consumers...")

	// todo gracefull shutdown

	for {
		select {
		case d := <-app.queue.UploadConsumer:
			app.logger.Debug("Consume upload...")
			d.Ack(false)
		case d := <-app.queue.DeleteConsumer:
			app.logger.Debug("Consume delete...")
			d.Ack(false)
		case d := <-app.queue.ProcessConsumer:
			app.logger.Debug("Consume process...")
			d.Ack(false)
		}
	}

}
