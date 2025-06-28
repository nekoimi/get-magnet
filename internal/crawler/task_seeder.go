package crawler

import (
	"github.com/nekoimi/get-magnet/internal/crawler/providers/sehuatang"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

var (
	seeders = make([]task.Seeder, 0)
	crontab = cron.New()
)

func init() {
	//register(test.TaskSeeder())
	//register(javdb.TaskSeeder())
	register(sehuatang.TaskSeeder())
}

func startTaskSeeders() {
	for _, p := range seeders {
		log.Debugf("启动%s任务生成...\n", p.Name())
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
