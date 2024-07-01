package storage

import (
	"github.com/nekoimi/get-magnet/common/model"
)

type Type int

const (
	Console = iota
	File
	Db
)

type Storage interface {
	Save(item *model.Item) error
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
