package model

import (
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"time"
)

// Item 单个任务结果信息
type Item struct {
	Title       string   `json:"title,omitempty"`
	Number      string   `json:"number,omitempty"`
	OptimalLink string   `json:"optimal_link,omitempty"`
	Links       []string `json:"links,omitempty"`
	ResHost     string   `json:"res_host,omitempty"`
	ResPath     string   `json:"res_path,omitempty"`
}

// Magnet 磁力信息实体 magnets
type Magnet struct {
	Id        int       `json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    uint8     `json:"status,omitempty"`
	Item      `json:"item"`
}

func (m *Item) String() string {
	return util.ToJson(m)
}
