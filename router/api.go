package router

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
)

func New(builder *router.APIBuilder) {
	builder.Get("/", func(c *context.Context) {
		c.WriteString("Hello world!")
	})
	api := builder.Party("/api")
	v1Api := api.Party("/v1")
	{
		v1Api.Get("/login", func(ctx *context.Context) {
			ctx.WriteString("login!")
		})
	}
}
