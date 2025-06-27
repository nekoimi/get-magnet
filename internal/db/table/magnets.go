package table

import "time"

type Magnets struct {
	Id          int64     `json:"id,omitempty"`
	CreatedAt   time.Time `xorm:"created" json:"created_at"`
	UpdatedAt   time.Time `xorm:"updated" json:"updated_at"`
	Origin      string    `json:"origin,omitempty"`
	Title       string    `json:"title,omitempty"`
	Number      string    `json:"number,omitempty"`
	OptimalLink string    `json:"optimal_link,omitempty"`
	Links       []string  `json:"links,omitempty"`
	RawURLHost  string    `xorm:"raw_url_host" json:"raw_url_host,omitempty"`
	RawURLPath  string    `xorm:"raw_url_path" json:"raw_url_path,omitempty"`
	Status      uint8     `json:"status,omitempty"`
}
