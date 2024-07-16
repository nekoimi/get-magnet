package table

import (
	"strconv"
)

type Admin struct {
	Table
	Username string `json:"username,omitempty"`
	Password string `json:"-"`
}

func (a *Admin) GetId() string {
	return strconv.FormatInt(a.Id, 10)
}
