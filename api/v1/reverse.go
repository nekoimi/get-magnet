package v1

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewReverseAria2(jsonrpcUrl *url.URL) func(http.ResponseWriter, *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(jsonrpcUrl)
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		log.Printf("aria2接口代理错误: %s\n", err.Error())
		return
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		proxy.ServeHTTP(writer, request)
	}
}
