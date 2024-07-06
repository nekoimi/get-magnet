package router

import (
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/api/download"
	"github.com/nekoimi/get-magnet/middleware"
)

//const aria2JsonApi = "/api/aria2/jsonrpc"

//var (
//	r        *mux.Router
//	rwmux    sync.RWMutex
//	routeMap = make(map[string]func(http.ResponseWriter, *http.Request))
//)

func New() *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.AuthMiddleware())

	// 提交下载连接
	r.HandleFunc("/api/v1/download/submit", download.Submit)

	return r
}

//// Get 获取路由实例
//func Get() *mux.Router {
//	return r
//}

//// EnableAria2Reverse 开启aria2接口反向代理
//func EnableAria2Reverse(jsonrpcUrl *url.URL) {
//	rwmux.Lock()
//	defer rwmux.Unlock()
//	delete(routeMap, aria2JsonApi)
//
//	// 设置aria2接口代理，方便前端统一接口调用
//	routeMap[aria2JsonApi] = reverse.NewReverseAria2(jsonrpcUrl)
//	refreshRouter()
//	log.Printf("注册aria2接口代理路由: %s => %s \n", aria2JsonApi, jsonrpcUrl)
//}
//
//// DisableAria2Reverse 禁用aria2接口反向代理
//func DisableAria2Reverse() {
//	rwmux.Lock()
//	defer rwmux.Unlock()
//	delete(routeMap, aria2JsonApi)
//
//	refreshRouter()
//	log.Printf("取消注册aria2接口代理路由: %s\n", aria2JsonApi)
//}
//
//// 刷新路由
//func refreshRouter() {
//	r := mux.NewRouter()
//	r.Use(middleware.AuthMiddleware())
//	for path, handler := range routeMap {
//		r.HandleFunc(path, handler)
//		log.Printf("注册路由: %s\n", path)
//	}
//}
