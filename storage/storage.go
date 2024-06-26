package storage

import (
	"get-magnet/internal/model"
)

type Type int

const (
	Console = iota
	File
	Db
)

type Storage interface {
	Save(item *model.MagnetItem) error
}

func NewStorage(t Type) Storage {
	switch t {
	case File:
		return newFile("test")
	case Db:
		return newDb()
	default:
		return newConsole()
	}
}
