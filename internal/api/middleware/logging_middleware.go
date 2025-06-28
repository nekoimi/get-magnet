package middleware

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("request access: %s %s\n", r.Method, r.RequestURI)
		// next
		next.ServeHTTP(w, r)
	})
}
