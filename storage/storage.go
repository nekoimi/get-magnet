package storage

import (
	"get-magnet/scheduler"
)

type Storage interface {
	Save(item scheduler.MagnetItem) error
}
