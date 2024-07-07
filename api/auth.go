package api

import (
	"github.com/nekoimi/get-magnet/pkg/request"
	"github.com/nekoimi/get-magnet/pkg/response"
	"net/http"
)

type A struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	a := new(A)

	err := request.Parse(r, &a)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Ok(w, a)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	response.Ok(w, nil)
}
