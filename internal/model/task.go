package model

import "github.com/PuerkitoBio/goquery"

// Task 任务信息
type Task struct {
	Url    string
	Handle func(meta *TaskMeta, selection *goquery.Selection) (TaskOut, error)
	Meta   *TaskMeta
}

// TaskMeta 任务元信息
type TaskMeta struct {
	Host    string
	UrlPath string
}

// TaskOut 任务执行输出
type TaskOut struct {
	Tasks []Task
	Items []MagnetItem
}
