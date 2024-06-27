package storage

import (
	"get-magnet/internal/model"
	"log"
)

type consoleStorage struct {
}

func newConsole() Storage {
	return &consoleStorage{}
}

func (s *consoleStorage) Save(item *model.MagnetItem) error {
	log.Println(item)
	return nil
}