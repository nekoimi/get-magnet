package storage

import "get-magnet/engine"

type Storage interface {
	Save(item engine.MagnetItem) error
}
