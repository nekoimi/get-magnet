package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/jwt"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	log "github.com/sirupsen/logrus"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			if c, err := r.Cookie("token"); err == nil {
				token = c.Value
			} else {
				log.Debugf("获取请求cookie异常: %s", err.Error())
			}
		}

		if token == "" {
			respond.Error(w, error_ext.AuthenticationError)
			return
		}

		sub, err := jwt.ParseToken(token)
		if err != nil {
			if errors.Is(err, jwt.TokenExpireError) {
				respond.Error(w, error_ext.AuthenticationExpirseError)
			} else {
				respond.Error(w, error_ext.AuthenticationError)
			}
			return
		}

		authCtx := context.WithValue(r.Context(), request.ContextJwtUser, sub)
		next.ServeHTTP(w, r.WithContext(authCtx))
	})
}
