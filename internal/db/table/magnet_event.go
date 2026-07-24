package table

import "time"

type MagnetEvent struct {
	Id        int64     `json:"id,omitempty"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
	MagnetId  int64     `xorm:"magnet_id" json:"magnet_id"`
	EventType string    `xorm:"event_type" json:"event_type"`
	Message   string    `xorm:"text" json:"message,omitempty"`
	Extra     string    `xorm:"text" json:"extra,omitempty"`
}

func (MagnetEvent) TableName() string {
	return "magnet_events"
}
