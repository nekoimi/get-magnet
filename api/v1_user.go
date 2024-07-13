package api

import (
	"github.com/nekoimi/get-magnet/database"
	"github.com/nekoimi/get-magnet/database/table"
	"github.com/nekoimi/get-magnet/pkg/error_ext"
	"github.com/nekoimi/get-magnet/pkg/request"
	"github.com/nekoimi/get-magnet/pkg/respond"
	"github.com/nekoimi/get-magnet/pkg/util"
	"net/http"
)

// Me 获取当前登录用户信息
func Me(w http.ResponseWriter, r *http.Request) {
	u, ok := request.JwtUser(w, r)
	if !ok {
		return
	}

	respond.Ok(w, u)
}

type ChangePasswordRequest struct {
	Password        string `json:"password,omitempty"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
}

// ChangePassword 修改当前用户密码
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	u, ok := request.JwtUser(w, r)
	if !ok {
		return
	}

	p := new(ChangePasswordRequest)
	if err := request.Parse[ChangePasswordRequest](r, p); err != nil {
		respond.Error(w, error_ext.ValidateError)
		return
	}

	encodePassword, err := util.BcryptEncode(p.Password)
	if err != nil {
		respond.Error(w, err)
		return
	}

	if _, err := database.Instance().Table(new(table.Admin)).ID(u.GetId()).Cols("password").Update(map[string]interface{}{
		"password": encodePassword,
	}); err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, nil)
}
