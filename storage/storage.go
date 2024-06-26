package storage

import (
	"get-magnet/internal/model"
)

type Storage interface {
	Save(item model.MagnetItem) error
}
