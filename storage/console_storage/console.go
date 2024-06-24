package console_storage

import (
	"get-magnet/scheduler"
	"get-magnet/storage"
	"log"
)

type consoleStorage struct {
}

func New() storage.Storage {
	return &consoleStorage{}
}

func (s *consoleStorage) Save(item scheduler.MagnetItem) error {
	log.Println(item)
	return nil
}
