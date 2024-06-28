package storage

import (
	"github.com/nekoimi/get-magnet/internal/model"
	"log"
)

type consoleStorage struct {
}

func newConsole() Storage {
	return &consoleStorage{}
}

func (s *consoleStorage) Save(item *model.Item) error {
	log.Println(item)
	return nil
}
