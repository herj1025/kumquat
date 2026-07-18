package main

import (
	"log"

	"github.com/herj1025/kumquat"
	"github.com/herj1025/kumquat/pkg/server"
)

func main() {
	app := kumquat.NewApp()
	app.AddGenerateCommand()
	app.RegisterRoutes(func(app *server.Application) {
		if err := registerRoutes(app); err != nil {
			log.Fatal(err)
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
