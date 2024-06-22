package console_storage

import (
	"get-magnet/engine"
	"get-magnet/pkg/util"
	"get-magnet/storage"
	"log"
)

type consoleStorage struct {
}

func New() storage.Storage {
	return &consoleStorage{}
}

func (s *consoleStorage) Save(item engine.ParseItem) error {
	log.Println(util.ToJson(item))
	return nil
}
