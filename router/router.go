package router

import (
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/api/download"
	"github.com/nekoimi/get-magnet/middleware"
)

func New() *mux.Router {
	r := mux.NewRouter()

	r.Use(middleware.AuthMiddleware())

	// 提交下载连接
	r.HandleFunc("/api/v1/download/submit", download.Submit)

	return r
}
