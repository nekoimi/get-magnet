package server

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/internal/api/auth"
	"github.com/nekoimi/get-magnet/internal/api/crawler"
	"github.com/nekoimi/get-magnet/internal/api/dashboard"
	"github.com/nekoimi/get-magnet/internal/api/magnets"
	"github.com/nekoimi/get-magnet/internal/api/middleware"
	"github.com/nekoimi/get-magnet/internal/api/ops"
	"github.com/nekoimi/get-magnet/internal/api/resources"
	"github.com/nekoimi/get-magnet/internal/api/settings"
	"github.com/nekoimi/get-magnet/internal/api/ui"
	"github.com/nekoimi/get-magnet/internal/api/user"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	crawlercore "github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/job"
	log "github.com/sirupsen/logrus"
)

const uiDir = "/workspace/ui"

func newRouter(ctx context.Context, cfg *config.Config) *mux.Router {
	r := mux.NewRouter()
	cronScheduler := bean.FromContext[job.CronScheduler](ctx)
	crawlerEngine := bean.PtrFromContext[crawlercore.Engine](ctx)
	crawlerManager := bean.PtrFromContext[crawlercore.Manager](ctx)

	r.Use(middleware.CORSMiddleware)
	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).Methods("GET")

	// 无需认证的接口必须在受保护的 /api 子路由之前注册。
	r.HandleFunc("/api/auth/login", auth.Login)

	// 需要认证的接口
	apiRoute := r.PathPrefix("/api").Subrouter()
	apiRoute.Use(middleware.AuthMiddleware)
	{
		// 登出
		apiRoute.HandleFunc("/auth/logout", auth.Logout)

		v1Api := apiRoute.PathPrefix("/v1").Subrouter()
		{
			v1Api.HandleFunc("/dashboard/summary", dashboard.Summary).Methods("GET")
			v1Api.HandleFunc("/settings", settings.List(cfg)).Methods("GET")
			v1Api.HandleFunc("/settings/testDrissionRod", settings.TestDrissionRod(cfg)).Methods("POST")
			v1Api.HandleFunc("/ops/health", ops.Health(cfg)).Methods("GET")
			v1Api.HandleFunc("/ops/jobs", ops.Jobs(cronScheduler)).Methods("GET")
			v1Api.HandleFunc("/ops/version", ops.Version).Methods("GET")

			// 获取当前用户信息
			v1Api.HandleFunc("/me", user.Me)
			// 修改当前用户密码
			v1Api.HandleFunc("/me/changePwd", user.ChangePassword)
			v1Api.HandleFunc("/crawler/submit/javdb", crawlerapi.SubmitJavDB).Methods("POST")
			v1Api.HandleFunc("/crawler/submit/javdbPage", crawlerapi.SubmitJavDBPage).Methods("POST")
			v1Api.HandleFunc("/crawler/status", crawlerapi.Status(crawlerEngine)).Methods("GET")
			v1Api.HandleFunc("/crawler/providers", crawlerapi.Providers(crawlerManager)).Methods("GET")
			v1Api.HandleFunc("/crawler/run", crawlerapi.Run(crawlerManager)).Methods("POST")
			// 磁力链接管理
			v1Api.HandleFunc("/magnets/list", magnets.List).Methods("GET", "POST")
			v1Api.HandleFunc("/magnets/statusOptions", magnets.StatusOptions).Methods("GET")
			v1Api.HandleFunc("/magnets/sourceOptions", magnets.SourceOptions).Methods("GET")
			v1Api.HandleFunc("/magnets/detail", magnets.Detail(cfg)).Methods("GET")
			v1Api.HandleFunc("/magnets/create", magnets.Create).Methods("POST")
			v1Api.HandleFunc("/magnets/update", magnets.Update).Methods("POST")
			v1Api.HandleFunc("/magnets/delete", magnets.Delete).Methods("POST")
			v1Api.HandleFunc("/magnets/markStatus", magnets.MarkStatus).Methods("POST")
		}

		v2Api := apiRoute.PathPrefix("/v2").Subrouter()
		{
			v2Api.HandleFunc("/resources/list", resources.List).Methods("GET", "POST")
			v2Api.HandleFunc("/resources/detail", resources.Detail).Methods("GET")
			v2Api.HandleFunc("/resources/statusOptions", resources.StatusOptions).Methods("GET")
			v2Api.HandleFunc("/resources/sourceOptions", resources.SourceOptions).Methods("GET")
			v2Api.HandleFunc("/resources/create", resources.Create).Methods("POST")
			v2Api.HandleFunc("/resources/update", resources.Update).Methods("POST")
			v2Api.HandleFunc("/resources/delete", resources.Delete).Methods("POST")
			v2Api.HandleFunc("/resources/markStatus", resources.MarkStatus).Methods("POST")
		}
	}

	// 静态资源
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
