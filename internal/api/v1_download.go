package api

import (
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"net/http"
)

func Submit(w http.ResponseWriter, r *http.Request) {
	respond.Ok(w, nil)
}
