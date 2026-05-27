package main

import (
	"log"

	"github.com/herj1025/kumquat"
	"github.com/herj1025/kumquat/internal/demo"
	"github.com/herj1025/kumquat/pkg/server"
)

func main() {
	app := kumquat.NewApp()
	app.AddGenerateCommand()
	app.RegisterRoutes(func(app *server.Application) {
		repo := demo.NewRepository(app.Deps().DB())
		svc := demo.NewService(repo, app.Deps().Redis(), app.Deps().DistLock())
		handler := demo.NewHandler(svc)

		r := app.Engine()
		r.GET("/demo", handler.Get)
		r.POST("/demo/task", handler.RunTask)
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
