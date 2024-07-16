package table

type Migrates struct {
	Table
	Version int64  `json:"version,omitempty"`
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
}
