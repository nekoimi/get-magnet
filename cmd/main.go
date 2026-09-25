package main

import (
	"github.com/nekoimi/scrapio/internal/bootstrap"
	log "github.com/sirupsen/logrus"
)

func main() {
	// 初始化服务
	lifecycle := bootstrap.BeanLifecycle()
	// 启动服务
	if err := lifecycle.StartAndServe(); err != nil {
		log.Fatal(err)
	}
}
