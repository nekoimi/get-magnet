package router

import (
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/api"
	"github.com/nekoimi/get-magnet/api/v1"
	"github.com/nekoimi/get-magnet/middleware"
	"log"
	"net/http"
)

// const uiStaticDir = "/workspace/ui"
const uiStaticDir = "C:\\Users\\nekoimi\\Downloads\\vue-next-admin"

//const aria2JsonApi = "/api/aria2/jsonrpc"

//var (
//	r        *mux.Router
//	rwmux    sync.RWMutex
//	routeMap = make(map[string]func(http.ResponseWriter, *http.Request))
//)

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

		v1Api := apiRoute.PathPrefix("/v1").Subrouter()
		{
			// 提交下载连接
			v1Api.HandleFunc("/download/submit", v1.Submit)
		}
	}

	// 静态资源
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(uiStaticDir))))

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
