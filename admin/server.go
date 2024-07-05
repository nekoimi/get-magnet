package admin

import "net/http"

func NewServer() *http.Server {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: newRouter(),
	}

	return srv
}
