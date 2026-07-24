package server

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/internal/api/auth"
	"github.com/nekoimi/get-magnet/internal/api/crawler"
	"github.com/nekoimi/get-magnet/internal/api/dashboard"
	"github.com/nekoimi/get-magnet/internal/api/download"
	"github.com/nekoimi/get-magnet/internal/api/magnets"
	"github.com/nekoimi/get-magnet/internal/api/middleware"
	"github.com/nekoimi/get-magnet/internal/api/play"
	"github.com/nekoimi/get-magnet/internal/api/proxy"
	"github.com/nekoimi/get-magnet/internal/api/ui"
	"github.com/nekoimi/get-magnet/internal/api/user"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/downloader"
	log "github.com/sirupsen/logrus"
)

const uiDir = "/workspace/ui"
const uiAriaNgDir = "/workspace/ui/aria-ng"
const aria2JsonApi = "/api/aria2/jsonrpc"

func newRouter(ctx context.Context, cfg *config.Config) *mux.Router {
	r := mux.NewRouter()
	downloadService := bean.FromContext[downloader.DownloadService](ctx)

	r.Use(middleware.CORSMiddleware)
	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.AuthMiddleware())

	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).Methods("GET")

	// aria2 jsonrpc 代理
	r.HandleFunc(aria2JsonApi, proxy.ReverseAria2())
	// 接口
	apiRoute := r.PathPrefix("/api").Subrouter()
	{
		// 登录
		apiRoute.HandleFunc("/auth/login", auth.Login)
		// 登出
		apiRoute.HandleFunc("/auth/logout", auth.Logout)
		// 视频播放地址
		apiRoute.HandleFunc("/play/{number}", play.Play(cfg)).Methods("GET")

		v1Api := apiRoute.PathPrefix("/v1").Subrouter()
		{
			v1Api.HandleFunc("/dashboard/summary", dashboard.Summary).Methods("GET")

			// 获取当前用户信息
			v1Api.HandleFunc("/me", user.Me)
			// 修改当前用户密码
			v1Api.HandleFunc("/me/changePwd", user.ChangePassword)
			// 提交下载连接
			v1Api.HandleFunc("/download/submit", download.Submit(downloadService)).Methods("POST")
			v1Api.HandleFunc("/download/retry", download.Retry(downloadService)).Methods("POST")
			v1Api.HandleFunc("/crawler/submit/javdb", crawlerapi.SubmitJavDB).Methods("POST")
			v1Api.HandleFunc("/crawler/submit/javdbPage", crawlerapi.SubmitJavDBPage).Methods("POST")
			// 磁力链接管理
			v1Api.HandleFunc("/magnets/list", magnets.List).Methods("GET", "POST")
			v1Api.HandleFunc("/magnets/statusOptions", magnets.StatusOptions).Methods("GET")
			v1Api.HandleFunc("/magnets/detail", magnets.Detail).Methods("GET")
			v1Api.HandleFunc("/magnets/create", magnets.Create).Methods("POST")
			v1Api.HandleFunc("/magnets/update", magnets.Update).Methods("POST")
			v1Api.HandleFunc("/magnets/delete", magnets.Delete).Methods("POST")
		}
	}

	// 扩展接口
	r.HandleFunc("/quick-api/download/submit/javdb", download.SubmitJavDB)
	r.HandleFunc("/quick-api/download/submit/javdb_page", download.SubmitJavDBPage)
	//r.HandleFunc("/quick-api/download/submit/fc2", download.SubmitFC2)

	// 静态资源
	r.PathPrefix("/ui/aria-ng/").Handler(ui.Aria2WebUI(uiAriaNgDir))
	r.PathPrefix("/").Handler(ui.AdminUI(uiDir))

	debugRoute(r)

	return r
}

func debugRoute(r *mux.Router) {
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		log.Debugf("Route: %s", path)
		return nil
	})
}
