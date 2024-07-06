package api

import (
	"github.com/nekoimi/get-magnet/pkg/response"
	"net/http"
)

func Login(w http.ResponseWriter, request *http.Request) {
	response.Ok(w)
}

func Logout(w http.ResponseWriter, request *http.Request) {
	response.Ok(w)
}
