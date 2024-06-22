package engine

import "get-magnet/storage"

type Scheduler interface {
}

type SimpleScheduler struct {
	requestChan chan string

	// 存储接口
	storage *storage.Storage
}
