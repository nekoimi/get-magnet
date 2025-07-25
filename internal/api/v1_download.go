package api

import (
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
	//bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
	//	rawUrl,
	//	crawler.WithHandle(javdb2.TaskSeeder()),
	//	crawler.WithDownloader(javdb2.GetBypassDownloader()),
	//))

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
	//bus.Event().Publish(bus.SubmitTask.String(), crawler.NewCrawlerTask(
	//	rawUrl,
	//	crawler.WithHandle(sehuatang2.TaskSeeder()),
	//	crawler.WithDownloader(sehuatang2.GetBypassDownloader()),
	//))

	respond.Ok(w, nil)
}
