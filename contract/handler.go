package contract

import "github.com/PuerkitoBio/goquery"

// TaskHandler 任务处理器
type TaskHandler interface {
}

// SimpleTaskHandler 默认简单任务处理器
type SimpleTaskHandler interface {
	TaskHandler
	Handle(url string) ([]WorkerTask, any, error)
}

// HTMLQueryParseHandler 使用 goquery 的HTML页面解析处理器
type HTMLQueryParseHandler interface {
	TaskHandler
	Handle(s *goquery.Selection) ([]WorkerTask, any, error)
}
