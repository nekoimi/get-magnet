package dynamic

import (
	"github.com/dop251/goja"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"log"
)

type dynamicProvider struct {
}

func Handler() task.Handler {
	return &dynamicProvider{}
}

func (p *dynamicProvider) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	vm := goja.New()
	v, err := vm.RunString("1 + 1")
	if err != nil {
		return nil, nil, err
	}

	log.Println(v)
	return nil, nil, nil
}
