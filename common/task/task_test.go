package task

import (
	"net/url"
	"testing"
)

func TestNewTask(t *testing.T) {
	u, err := url.Parse("https://movie.douban.com/top250")
	if err != nil {
		panic(err)
	}

	t.Log(u.Scheme, " ", u.Host, " ", u.Path)

	task := NewTask("https://movie.douban.com/top250", nil)
	t.Log(task)
}
