package middleware

import (
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/pkg/error_ext"
	"github.com/nekoimi/get-magnet/pkg/jwt"
	"github.com/nekoimi/get-magnet/pkg/request"
	"github.com/nekoimi/get-magnet/pkg/response"
	"log"
	"net/http"
)

var allowRequestUriMap = make(map[string]struct{})

func AuthMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uri := r.RequestURI
			if _, ok := allowRequestUriMap[uri]; ok {
				next.ServeHTTP(w, r)
				return
			}

			token := r.Header.Get("Authorization")
			if token == "" {
				if c, err := r.Cookie("token"); err != nil {
					log.Printf("获取请求cookie异常: %s\n", err.Error())
				} else {
					token = c.Value
				}
			}

			if sub, err := jwt.ParseToken(token); err != nil {
				if errors.Is(err, jwt.TokenExpirseError) {
					response.Error(w, error_ext.AuthenticationExpirseError)
				} else {
					response.Error(w, err)
				}
				return
			} else {
				// 将用户信息放到Context传递下去
				authCtx := context.WithValue(r.Context(), request.ContextJwtUser, sub)
				r.WithContext(authCtx)
			}

			// next
			next.ServeHTTP(w, r)
		})
	}
}
