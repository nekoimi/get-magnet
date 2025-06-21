package request

import (
	"encoding/json"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/jwt"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const ContextJwtUser = "request.context.jwtUser"

// JwtUser 从请求上下文解析jwt用户信息
func JwtUser(w http.ResponseWriter, r *http.Request) (jwt.Subject, bool) {
	if subId, ok := r.Context().Value(ContextJwtUser).(string); !ok {
		respond.Error(w, error_ext.AuthenticationError)
		return nil, false
	} else {
		id, err := strconv.ParseInt(subId, 10, 64)
		if err != nil {
			respond.Error(w, err)
			return nil, false
		}
		u := new(table.Admin)
		u.Id = id
		if has, err := db.Instance().Get(u); err != nil {
			respond.Error(w, err)
			return nil, false
		} else if !has {
			respond.Error(w, error_ext.AuthenticationError)
			return nil, false
		}
		return u, true
	}
}

func Parse[T any](r *http.Request, t *T) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return error_ext.RequestBodyNotSupportedError
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, t)
	if err != nil {
		return err
	}
	return nil
}
