package api

import (
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/sehuatang"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func Submit(w http.ResponseWriter, r *http.Request) {
	respond.Ok(w, nil)
}

type TaskReq struct {
	Url string `json:"url,omitempty"`
}

func SubmitJavDB(w http.ResponseWriter, r *http.Request) {
	p := new(TaskReq)

	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	} else {
		if p.Url == "" {
			respond.Error(w, error_ext.ValidateError)
			return
		}
	}

	rawUrl := p.Url
	log.Infof("接收到JavDB链接任务：%s", rawUrl)
	bus.Event().Publish(bus.SubmitTask.String(), task.NewTask(rawUrl, task.WithHandle(javdb.TaskSeeder())))

	respond.Ok(w, nil)
}

func SubmitFC2(w http.ResponseWriter, r *http.Request) {
	p := new(TaskReq)

	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	} else {
		if p.Url == "" {
			respond.Error(w, error_ext.ValidateError)
			return
		}
	}

	rawUrl := p.Url
	log.Infof("接收到FC2链接任务：%s", rawUrl)
	bus.Event().Publish(bus.SubmitTask.String(), task.NewTask(
		rawUrl,
		task.WithHandle(sehuatang.TaskSeeder()),
		task.WithDownloader(sehuatang.GetBypassDownloader()),
	))

	respond.Ok(w, nil)
}
