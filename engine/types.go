package engine

import (
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
	Items []any
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
	Id        int
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    uint8
	MagnetItem
}
