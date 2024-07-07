package middleware

import (
	"log"
	"net/http"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request access: %s %s\n", r.Method, r.RequestURI)
		// next
		next.ServeHTTP(w, r)
	})
}
