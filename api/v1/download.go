package v1

import (
	"github.com/nekoimi/get-magnet/pkg/response"
	"net/http"
)

func Submit(w http.ResponseWriter, request *http.Request) {
	response.Ok(w)
}
