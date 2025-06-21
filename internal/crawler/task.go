package crawler

import (
	"net/url"
)

type Task interface {
	Url() string
}

type WorkerTask interface {
	RawUrl() string
	ErrorNum() int
	IncrErrorNum()
	Handler() WorkerTaskHandler
}

type Aria2Task struct {
	Url string
}

// Magnet 任务结果信息
type Magnet struct {
	Title       string   `json:"title,omitempty"`
	Number      string   `json:"number,omitempty"`
	OptimalLink string   `json:"optimal_link,omitempty"`
	Links       []string `json:"links,omitempty"`
	ResHost     string   `json:"res_host,omitempty"`
	ResPath     string   `json:"res_path,omitempty"`
}

type TaskEntry struct {
	RawURL     string `json:"raw_url,omitempty"`
	RawURLHost string `json:"raw_url_host,omitempty"`
	RawURLPath string `json:"raw_url_path,omitempty"`
	ErrorCount int    `json:"error_count,omitempty"`
	handle     WorkerTaskHandler
	downloader Downloader
}

// NewWorkerTask 创建默认任务实体
func NewWorkerTask(rawURL string, handle WorkerTaskHandler) WorkerTask {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	return &TaskEntry{
		RawURL:     rawURL,
		RawURLHost: u.Scheme + "://" + u.Host,
		RawURLPath: u.Path,
		ErrorCount: 0,
		handle:     handle,
		downloader: NewDefaultDownloader(),
	}
}

func (t *TaskEntry) RawUrl() string {
	return t.RawURL
}

func (t *TaskEntry) ErrorNum() int {
	return t.ErrorCount
}

func (t *TaskEntry) IncrErrorNum() {
	t.ErrorCount = t.ErrorCount + 1
}

func (t *TaskEntry) Handler() WorkerTaskHandler {
	return t.handle
}

func (t *TaskEntry) Downloader() Downloader {
	return t.downloader
}
