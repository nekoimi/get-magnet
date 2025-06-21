package dynamic

import (
	"github.com/dop251/goja"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"log"
)

type dynamicProvider struct {
}

func Handler() crawler.WorkerTaskHandler {
	return &dynamicProvider{}
}

func (p *dynamicProvider) Handle(t crawler.WorkerTask) (tasks []crawler.WorkerTask, outputs []crawler.Magnet, err error) {
	vm := goja.New()
	v, err := vm.RunString("1 + 1")
	if err != nil {
		return nil, nil, err
	}

	log.Println(v)
	return nil, nil, nil
}
