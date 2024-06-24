package scheduler

import (
	"get-magnet/pkg/util"
	"github.com/PuerkitoBio/goquery"
	"time"
)

// Task 任务信息
type Task struct {
	Url    string
	Handle func(selection *goquery.Selection) (TaskOut, error)
}

// TaskOut 任务执行输出
type TaskOut struct {
	Tasks []Task
	Items []MagnetItem
}

// MagnetItem 单个任务结果信息
type MagnetItem struct {
	Title       string   `json:"title,omitempty"`
	Number      string   `json:"number,omitempty"`
	OptimalLink string   `json:"optimal_link,omitempty"`
	Links       []string `json:"links,omitempty"`
	ResHost     string   `json:"res_host,omitempty"`
	ResPath     string   `json:"res_path,omitempty"`
}

// Magnet 磁力信息实体 magnets
type Magnet struct {
	Id         int       `json:"id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Status     uint8     `json:"status,omitempty"`
	MagnetItem `json:"magnet_item"`
}

func (m *MagnetItem) String() string {
	return util.ToJson(m)
}
