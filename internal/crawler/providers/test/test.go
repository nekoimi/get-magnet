package test

import (
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	log "github.com/sirupsen/logrus"
	"time"
)

type Seeder struct {
}

var (
	// seeder实例
	seederSingleton = singleton.New[*Seeder](func() *Seeder {
		return &Seeder{}
	})
)

func TaskSeeder() *Seeder {
	return seederSingleton.Get()
}

func (p *Seeder) Name() string {
	return "Test"
}

func (p *Seeder) Exec() {
	timer := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-timer.C:
			bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://www.baidu.com/", task.WithHandle(TaskSeeder())))
			log.Infof("启动任务：%s", p.Name())
		}
	}
}

func (p *Seeder) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理任务：%s\n", rawUrl)
		time.Sleep(10 * time.Second)
		return nil, nil, nil
	}
	return nil, nil, nil
}
