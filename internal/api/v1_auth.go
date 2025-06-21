package api

import (
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/jwt"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"log"
	"net/http"
)

type LoginReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Login 登录认证
func Login(w http.ResponseWriter, r *http.Request) {
	p := new(LoginReq)

	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	} else {
		if p.Username == "" || p.Password == "" {
			respond.Error(w, error_ext.ValidateError)
			return
		}
	}

	admin := &table.Admin{Username: p.Username}
	if has, err := db.Instance().Get(admin); err != nil {
		respond.Error(w, err)
		return
	} else if !has {
		respond.Error(w, error_ext.AccountNotFoundError)
		return
	}

	if !util.Check(admin.Password, p.Password) {
		respond.Error(w, error_ext.PasswordError)
		return
	}

	result, err := jwt.NewToken(admin)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, result)
}

// Logout 退出登录
func Logout(w http.ResponseWriter, r *http.Request) {
	u, ok := request.JwtUser(w, r)
	if !ok {
		return
	}

	log.Printf("登录用户: %s\n", u.GetId())

	respond.Ok(w, nil)
}
