package table

import "time"

type Migrates struct {
	Id        int64     `json:"id,omitempty"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
	DeletedAt time.Time `xorm:"deleted" json:"-"`
	Version   int64     `json:"version,omitempty"`
	Success   bool      `json:"success,omitempty"`
	Message   string    `json:"message,omitempty"`
}
