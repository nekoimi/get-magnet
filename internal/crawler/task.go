package crawler

import (
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"net/url"
)

type TaskHandler func(t CrawlerTask) ([]CrawlerTask, []MagnetEntry, error)

type CrawlerTask interface {
	RawOrigin() string
	RawUrl() string
	ErrorNum() int
	IncrErrorNum()
	Handler() TaskHandler
}

type Option func(t *TaskEntry)

// TaskEntry 任务信息
type TaskEntry struct {
	// 任务信息
	Origin     string
	RawURL     string
	RawURLHost string
	RawURLPath string
	ErrorCount int
	handle     TaskHandler
	downloader download.Downloader
}

// MagnetEntry 任务结果信息
type MagnetEntry struct {
	Origin      string   `json:"origin,omitempty"`
	Title       string   `json:"title,omitempty"`
	Number      string   `json:"number,omitempty"`
	Actress0    string   `json:"actress0,omitempty"`
	OptimalLink string   `json:"optimal_link,omitempty"`
	Links       []string `json:"links,omitempty"`
	RawURLHost  string   `json:"raw_url_host,omitempty"`
	RawURLPath  string   `json:"raw_url_path,omitempty"`
}

// TorrentLink 磁力链接信息
type TorrentLink struct {
	Sort int
	Name string
	Link string
}

// NewCrawlerTask 创建默认任务实体
func NewCrawlerTask(rawURL string, origin string, opts ...Option) CrawlerTask {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	t := &TaskEntry{
		Origin:     origin,
		RawURL:     rawURL,
		RawURLHost: u.Scheme + "://" + u.Host,
		RawURLPath: u.RequestURI(),
		ErrorCount: 0,
		downloader: download.NewHttpDownloader(),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (t *TaskEntry) RawOrigin() string {
	return t.Origin
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

func (t *TaskEntry) Handler() TaskHandler {
	return t.handle
}

func (t *TaskEntry) Downloader() download.Downloader {
	return t.downloader
}

func WithHandle(handle TaskHandler) Option {
	return func(t *TaskEntry) {
		t.handle = handle
	}
}

func WithDownloader(downloader download.Downloader) Option {
	return func(t *TaskEntry) {
		t.downloader = downloader
	}
}
