package crawler

import (
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/robfig/cron/v3"
	"log"
)

var (
	seeders = make([]task.Seeder, 0)
	crontab = cron.New()
)

func init() {
	register(&javdb.Seeder{})
}

func startTaskSeeders() {
	for _, p := range seeders {
		log.Printf("启动%s任务生成...\n", p.Name())
		p.Exec(crontab)
	}

	go crontab.Start()
}

func stopTaskSeeders() {
	<-crontab.Stop().Done()
}

func register(p task.Seeder) {
	seeders = append(seeders, p)
}
