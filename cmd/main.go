package main

import (
	"github.com/nekoimi/get-magnet/internal/bootstrap"
)

func main() {
	// 初始化服务
	lifecycle := bootstrap.BeanLifecycle()
	// 启动服务
	lifecycle.StartAndServe()
}
