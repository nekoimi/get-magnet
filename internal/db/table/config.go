package table

import "time"

type Config struct {
	Id        int64     `json:"id,omitempty"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
	DeletedAt time.Time `xorm:"deleted" json:"-"`
	// 配置类型
	Type string
	// 配置key
	Key string
	// 配置value
	Value string
}
