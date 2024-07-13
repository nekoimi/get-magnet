package middleware

import (
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/pkg/error_ext"
	"github.com/nekoimi/get-magnet/pkg/jwt"
	"github.com/nekoimi/get-magnet/pkg/request"
	"github.com/nekoimi/get-magnet/pkg/respond"
	"log"
	"net/http"
	"strings"
)

var allowRequestUriMap = make(map[string]struct{})

func init() {
	allowRequestUriMap["/ui/aria-ng"] = struct{}{}
	allowRequestUriMap["/api/auth/login"] = struct{}{}
}

func AuthMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uri := r.RequestURI
			for path := range allowRequestUriMap {
				if uri == path || strings.HasPrefix(uri, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			token := r.Header.Get("Authorization")
			if token == "" {
				if c, err := r.Cookie("token"); err == nil {
					token = c.Value
				} else {
					log.Printf("获取请求cookie异常: %s\n", err.Error())
				}
			}

			if token == "" {
				respond.Error(w, error_ext.AuthenticationError)
				return
			}

			if sub, err := jwt.ParseToken(token); err != nil {
				if errors.Is(err, jwt.TokenExpireError) {
					respond.Error(w, error_ext.AuthenticationExpirseError)
				} else {
					respond.Error(w, err)
				}
				return
			} else {
				// 将用户信息放到Context传递下去

				authCtx := context.WithValue(r.Context(), request.ContextJwtUser, sub)
				r = r.WithContext(authCtx)
			}

			// next
			next.ServeHTTP(w, r)
		})
	}
}
