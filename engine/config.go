package engine

import "github.com/nekoimi/get-magnet/storage"

type Config struct {
	WorkerNum int
	DbDsn     string
	AriaRpc   string
	AriaToken string
	Storage   storage.Type
}
