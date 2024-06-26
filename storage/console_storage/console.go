package console_storage

import (
	"get-magnet/internal/model"
	"get-magnet/storage"
	"log"
)

type consoleStorage struct {
}

func New() storage.Storage {
	return &consoleStorage{}
}

func (s *consoleStorage) Save(item model.MagnetItem) error {
	log.Println(item)
	return nil
}
