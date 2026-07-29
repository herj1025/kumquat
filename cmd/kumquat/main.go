package main

import (
	"log"

	"github.com/herj1025/kumquat"
)

func main() {
	app := kumquat.NewApp()
	app.AddGenerateCommand()
	app.RegisterRoutes(registerRoutes)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
