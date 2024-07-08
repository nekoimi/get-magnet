package router

import (
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/api"
	"github.com/nekoimi/get-magnet/api/v1"
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/middleware"
	"log"
)

const aria2JsonApi = "/api/aria2/jsonrpc"

func New() *mux.Router {
	r := mux.NewRouter()
	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.AuthMiddleware())
	// 接口
	apiRoute := r.PathPrefix("/api").Subrouter()
	{
		// 登录
		apiRoute.HandleFunc("/auth/login", api.Login)
		// 登出
		apiRoute.HandleFunc("/auth/logout", api.Logout)

		// aria2 jsonrpc 代理
		apiRoute.HandleFunc(aria2JsonApi, api.ReverseAria2())

		v1Api := apiRoute.PathPrefix("/v1").Subrouter()
		{
			// 提交下载连接
			v1Api.HandleFunc("/download/submit", v1.Submit)
		}
	}

	// 静态资源
	r.PathPrefix("/ui/aria-ng/").Handler(api.Aria2WebUI(config.UIAriaNgDir))
	r.PathPrefix("/").Handler(api.AdminUI(config.UIDir))

	routeDebugPrint(r)

	return r
}

func routeDebugPrint(r *mux.Router) {
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		log.Printf("Route: %s\n", path)
		return nil
	})
}

