package middleware

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithField("request_id", RequestID(r.Context())).Debugf("request access: %s %s", r.Method, r.RequestURI)
		// next
		next.ServeHTTP(w, r)
	})
}
