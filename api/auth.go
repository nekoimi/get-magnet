package api

import (
	"github.com/nekoimi/get-magnet/database"
	"github.com/nekoimi/get-magnet/database/table"
	"github.com/nekoimi/get-magnet/pkg/error_ext"
	"github.com/nekoimi/get-magnet/pkg/jwt"
	"github.com/nekoimi/get-magnet/pkg/request"
	"github.com/nekoimi/get-magnet/pkg/response"
	"github.com/nekoimi/get-magnet/pkg/util"
	"net/http"
)

type LoginReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	p := new(LoginReq)

	if err := request.Parse(r, &p); err != nil {
		response.Error(w, err)
		return
	} else {
		if p.Username == "" || p.Password == "" {
			response.Error(w, error_ext.ValidateError)
			return
		}
	}

	admin := &table.Admin{Username: p.Username}
	if has, err := database.Instance().Get(admin); err != nil {
		response.Error(w, err)
		return
	} else if !has {
		response.Error(w, error_ext.AccountNotFoundError)
		return
	}

	if !util.Check(admin.Password, p.Password) {
		response.Error(w, error_ext.PasswordError)
		return
	}

	result, err := jwt.NewToken(admin)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Ok(w, result)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	response.Ok(w, nil)
}
