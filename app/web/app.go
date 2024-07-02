package web

import (
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/nekoimi/get-magnet/router"
	"time"
)

func New() *iris.Application {
	app := iris.New()
	app.Use(iris.Compression)
	app.UseRouter(recover.New())
	app.Use(iris.LimitRequestBodySize(32 << 20))
	app.Configure(iris.WithTimeFormat(time.DateTime))
	app.Configure(iris.WithLogLevel(golog.DebugLevel.String()))
	router.New(app.APIBuilder)
	return app
}
