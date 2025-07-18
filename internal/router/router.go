package router

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/internal/api"
	"github.com/nekoimi/get-magnet/internal/api/middleware"
	"github.com/nekoimi/get-magnet/internal/config"
	log "github.com/sirupsen/logrus"
	"net/http"
)

const aria2JsonApi = "/api/aria2/jsonrpc"

func HttpServer(port int) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: newRouter(),
	}
}

func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.AuthMiddleware())

	// aria2 jsonrpc 代理
	r.HandleFunc(aria2JsonApi, api.ReverseAria2())
	// 接口
	apiRoute := r.PathPrefix("/api").Subrouter()
	{
		// 登录
		apiRoute.HandleFunc("/auth/login", api.Login)
		// 登出
		apiRoute.HandleFunc("/auth/logout", api.Logout)

		v1Api := apiRoute.PathPrefix("/v1").Subrouter()
		{
			// 获取当前用户信息
			v1Api.HandleFunc("/me", api.Me)
			// 修改当前用户密码
			v1Api.HandleFunc("/me/changePwd", api.ChangePassword)
			// 提交下载连接
			v1Api.HandleFunc("/download/submit", api.Submit)
		}
	}

	// 扩展接口
	r.HandleFunc("/quick-api/download/submit/javdb", api.SubmitJavDB)
	r.HandleFunc("/quick-api/download/submit/fc2", api.SubmitFC2)

	// 静态资源
	r.PathPrefix("/ui/aria-ng/").Handler(api.Aria2WebUI(config.UIAriaNgDir))
	r.PathPrefix("/").Handler(api.AdminUI(config.UIDir))

	routeDebugPrint(r)

	return r
}

func routeDebugPrint(r *mux.Router) {
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		log.Debugf("Route: %s", path)
		return nil
	})
}
