package middleware

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("request access: %s %s", r.Method, r.RequestURI)
		// next
		next.ServeHTTP(w, r)
	})
}
