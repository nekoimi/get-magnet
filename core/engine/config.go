package engine

import "github.com/nekoimi/get-magnet/storage"

type Config struct {
	WorkerNum int
	DbDsn     string
	Jsonrpc   string
	Secret    string
	Storage   storage.Type
}
