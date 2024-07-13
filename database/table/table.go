package table

import "time"

type Table struct {
	Id        int64     `json:"id,omitempty"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
	DeletedAt time.Time `xorm:"deleted" json:"-"`
}
