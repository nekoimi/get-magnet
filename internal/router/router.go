package router

import (
	"github.com/gorilla/mux"
	api2 "github.com/nekoimi/get-magnet/internal/api"
	middleware2 "github.com/nekoimi/get-magnet/internal/api/middleware"
	"github.com/nekoimi/get-magnet/internal/config"
	"log"
)

const aria2JsonApi = "/api/aria2/jsonrpc"

func New() *mux.Router {
	r := mux.NewRouter()
	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(middleware2.LoggingMiddleware)
	r.Use(middleware2.AuthMiddleware())

	// aria2 jsonrpc 代理
	r.HandleFunc(aria2JsonApi, api2.ReverseAria2())
	// 接口
	apiRoute := r.PathPrefix("/api").Subrouter()
	{
		// 登录
		apiRoute.HandleFunc("/auth/login", api2.Login)
		// 登出
		apiRoute.HandleFunc("/auth/logout", api2.Logout)

		v1Api := apiRoute.PathPrefix("/v1").Subrouter()
		{
			// 获取当前用户信息
			v1Api.HandleFunc("/me", api2.Me)
			// 修改当前用户密码
			v1Api.HandleFunc("/me/changePwd", api2.ChangePassword)
			// 提交下载连接
			v1Api.HandleFunc("/download/submit", api2.Submit)
		}
	}

	// 静态资源
	r.PathPrefix("/ui/aria-ng/").Handler(api2.Aria2WebUI(config.UIAriaNgDir))
	r.PathPrefix("/").Handler(api2.AdminUI(config.UIDir))

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
