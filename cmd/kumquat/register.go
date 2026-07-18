package main

import (
	"github.com/herj1025/kumquat/internal/demo"
	"github.com/herj1025/kumquat/pkg/server"
)

func registerRoutes(app *server.Application) error {
	if err := demo.RegisterRoutes(app); err != nil {
		return err
	}
	return nil
}
