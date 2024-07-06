package contract

import "github.com/PuerkitoBio/goquery"

type Task interface {
	Url() string
	Category() string
}

type DownloadTask interface {
	Task
}

type WorkerTask interface {
	Task
	GetHandler() TaskHandler
}

type TaskResult struct {
}

// TaskHandler 任务处理器
type TaskHandler interface {
}

// SimpleTaskHandler 默认简单任务处理器
type SimpleTaskHandler interface {
	TaskHandler
	Handle(url string)
}

// HTMLQueryParseHandler 使用 goquery 的HTML页面解析处理器
type HTMLQueryParseHandler interface {
	TaskHandler
	Handle(s *goquery.Selection)
}
