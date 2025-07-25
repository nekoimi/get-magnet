package dynamic

import (
	"github.com/dop251/goja"
	"github.com/nekoimi/get-magnet/internal/crawler"
	log "github.com/sirupsen/logrus"
)

func Handle(t crawler.CrawlerTask) (tasks []crawler.CrawlerTask, outputs []crawler.MagnetEntry, err error) {
	vm := goja.New()
	v, err := vm.RunString("1 + 1")
	if err != nil {
		return nil, nil, err
	}

	log.Debugln(v)
	return nil, nil, nil
}
