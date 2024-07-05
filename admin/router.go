package admin

import (
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/admin/api/download"
)

func newRouter() *mux.Router {
	r := mux.NewRouter()

	// 提交下载连接
	r.HandleFunc("/api/v1/download/submit", download.Submit)

	return r
}
