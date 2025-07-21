package task

import (
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"net/url"
)

type Seeder interface {
	Name() string
	Exec()
}

// Handler 任务处理器
type Handler interface {
	// Handle worker任务处理器
	Handle(t Task) ([]Task, []MagnetEntry, error)
}

type Task interface {
	RawUrl() string
	ErrorNum() int
	IncrErrorNum()
	Handler() Handler
}

type Option func(t *Entry)

// Entry 任务信息
type Entry struct {
	// 任务ID
	TaskId string
	// 是否是动态处理的任务
	IsDynamic bool
	// 任务信息
	RawURL     string
	RawURLHost string
	RawURLPath string
	ErrorCount int
	handle     Handler
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

// NewTask 创建默认任务实体
func NewTask(rawURL string, opts ...Option) Task {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	t := &Entry{
		TaskId:     "",
		IsDynamic:  false,
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

func (t *Entry) RawUrl() string {
	return t.RawURL
}

func (t *Entry) ErrorNum() int {
	return t.ErrorCount
}

func (t *Entry) IncrErrorNum() {
	t.ErrorCount = t.ErrorCount + 1
}

func (t *Entry) Handler() Handler {
	return t.handle
}

func (t *Entry) Downloader() download.Downloader {
	return t.downloader
}

func WithHandle(handle Handler) Option {
	return func(t *Entry) {
		t.handle = handle
	}
}

func WithDownloader(downloader download.Downloader) Option {
	return func(t *Entry) {
		t.downloader = downloader
	}
}
