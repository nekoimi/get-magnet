package crawlerapi

import (
	"net/http"

	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	log "github.com/sirupsen/logrus"
)

type SubmitRequest struct {
	URL string `json:"url,omitempty"`
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
