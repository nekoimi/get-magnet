package v1

import (
	"github.com/nekoimi/get-magnet/pkg/respond"
	"net/http"
)

func Submit(w http.ResponseWriter, r *http.Request) {
	respond.Ok(w, nil)
}
