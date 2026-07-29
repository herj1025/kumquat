package demo

import "github.com/herj1025/kumquat/pkg/server"

// RegisterRoutes 注册 demo 模块路由，并在模块内部完成依赖组装。
func RegisterRoutes(app *server.Application) error {
	db, err := app.Deps().DB()
	if err != nil {
		return err
	}
	distLock, err := app.Deps().DistLock()
	if err != nil {
		return err
	}

	svc := NewService(newRepository(db), distLock)
	// 将 svc 同时作为 demoFinder 和 demoTaskRunner 注入
	handler := newHandler(svc)

	routes := app.Engine().Group("/demo")
	routes.GET("", handler.Get)
	routes.POST("/task", handler.RunTask)

	return nil
}
