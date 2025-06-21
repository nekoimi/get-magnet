package table

import "time"

type Magnets struct {
	Id          int64     `json:"id,omitempty"`
	CreatedAt   time.Time `xorm:"created" json:"created_at"`
	UpdatedAt   time.Time `xorm:"updated" json:"updated_at"`
	Title       string    `json:"title,omitempty"`
	Number      string    `json:"number,omitempty"`
	OptimalLink string    `json:"optimal_link,omitempty"`
	Links       []string  `json:"links,omitempty"`
	ResHost     string    `json:"res_host,omitempty"`
	ResPath     string    `json:"res_path,omitempty"`
	Status      uint8     `json:"status,omitempty"`
}
