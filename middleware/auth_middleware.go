package middleware

import (
	"github.com/gorilla/mux"
	"net/http"
)

func AuthMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// TODO 检查JWT进行验权

			// next
			next.ServeHTTP(writer, request)
		})
	}
}
