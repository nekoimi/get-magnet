package crawler

import (
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/sehuatang"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	log "github.com/sirupsen/logrus"
)

var (
	seeders = make([]task.Seeder, 0)
)

func init() {
	//register(test.TaskSeeder())
	register(javdb.TaskSeeder())
	register(sehuatang.TaskSeeder())
}

func startTaskSeeders() {
	for _, p := range seeders {
		log.Debugf("启动%s任务生成...\n", p.Name())
		p.Exec()
	}
}

func register(p task.Seeder) {
	seeders = append(seeders, p)
}
