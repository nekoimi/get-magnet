package task

import (
	"get-magnet/internal/model"
	"get-magnet/pkg/util"
	"github.com/PuerkitoBio/goquery"
	"net/url"
)

type Handle func(meta *Meta, selection *goquery.Selection) (*Out, error)

// Task Task details info
type Task struct {
	ErrorCount   int    `json:"retry_count,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Url          string `json:"url,omitempty"`
	Handle       Handle `json:"-"`
	Meta         *Meta  `json:"meta,omitempty"`
}

// Meta Task meta info
type Meta struct {
	Host    string `json:"host,omitempty"`
	UrlPath string `json:"url_path,omitempty"`
}

func NewTask(urlStr string, handle Handle) *Task {
	u, err := url.Parse(urlStr)
	if err != nil {
		panic(err)
	}

	return &Task{
		ErrorCount: 0,
		Url:        urlStr,
		Handle:     handle,
		Meta: &Meta{
			Host:    u.Scheme + "://" + u.Host,
			UrlPath: u.Path,
		},
	}
}

func (t *Task) IncrError() {
	t.ErrorCount++
}

func (t *Task) SetErrorMessage(err string) {
	t.ErrorMessage = err
	t.IncrError()
}

func (t *Task) String() string {
	return util.ToJson(t)
}

// Out 任务执行输出
type Out struct {
	Tasks []*Task
	Items []*model.MagnetItem
}

func NewEmptyOut() *Out {
	return new(Out)
}

func NewSingleOut(t *Task, item *model.MagnetItem) *Out {
	var tasks []*Task
	if t != nil {
		tasks = append(tasks, t)
	}
	var items []*model.MagnetItem
	if item != nil {
		items = append(items, item)
	}
	return NewOut(tasks, items)
}

func NewOut(tasks []*Task, items []*model.MagnetItem) *Out {
	return &Out{
		Tasks: tasks,
		Items: items,
	}
}
