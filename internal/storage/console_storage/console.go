package console_storage

import (
	"get-magnet/internal/storage"
	"log"
)

type consoleStorage struct {
}

func New() storage.Storage {
	return &consoleStorage{}
}

func (s *consoleStorage) Save(magnetLink string) error {
	log.Println(magnetLink)
	return nil
}