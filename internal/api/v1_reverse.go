package api

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
)

func ReverseAria2() func(http.ResponseWriter, *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(nil)
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		log.Debugf("aria2接口代理错误: %s", err.Error())
		return
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		proxy.ServeHTTP(writer, request)
	}
}
