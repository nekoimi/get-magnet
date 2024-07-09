package table

import (
	"strconv"
	"time"
)

type Admin struct {
	Id        int64     `json:"id,omitempty"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
	DeletedAt time.Time `xorm:"deleted" json:"-"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"-"`
}

func (a *Admin) GetId() string {
	return strconv.FormatInt(a.Id, 10)
}
