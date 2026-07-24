package crawlerapi

import (
	"net/http"

	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	log "github.com/sirupsen/logrus"
)

type SubmitRequest struct {
	URL string `json:"url,omitempty"`
}

type RunRequest struct {
	Name string `json:"name,omitempty"`
}

func SubmitJavDB(w http.ResponseWriter, r *http.Request) {
	submit(w, r, bus.SubmitJavDB.Topic(), "JavDB")
}

func SubmitJavDBPage(w http.ResponseWriter, r *http.Request) {
	submit(w, r, bus.SubmitJavDBPage.Topic(), "JavDB-Page")
}

func submit(w http.ResponseWriter, r *http.Request, topic string, name string) {
	p := new(SubmitRequest)
	if err := request.Parse(r, p); err != nil {
		respond.Error(w, err)
		return
	}
	if p.URL == "" {
		respond.Error(w, error_ext.ValidateError)
		return
	}

	log.Infof("接收到%s采集任务：%s", name, p.URL)
	bus.Event().Publish(topic, p.URL)
	respond.Ok(w, nil)
}

func Status(engine *crawler.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respond.Ok(w, engine.Snapshot())
	}
}

func Providers(manager *crawler.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respond.Ok(w, manager.Providers())
	}
}

func Run(manager *crawler.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := new(RunRequest)
		if err := request.Parse(r, req); err != nil {
			respond.Error(w, err)
			return
		}
		if err := manager.Run(req.Name); err != nil {
			respond.Error(w, err)
			return
		}
		respond.Ok(w, map[string]string{"message": "采集任务已触发"})
	}
}
