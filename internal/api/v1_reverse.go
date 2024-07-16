package api

import (
	"log"
	"net/http"
	"net/http/httputil"
)

func ReverseAria2() func(http.ResponseWriter, *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(nil)
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		log.Printf("aria2接口代理错误: %s\n", err.Error())
		return
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		proxy.ServeHTTP(writer, request)
	}
}
