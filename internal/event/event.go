package event

import (
	"github.com/nekoimi/get-magnet/internal/bus"
)

type Type int

const (
	ScaleWorker Type = iota
	Download
	Aria2Test
	Aria2LinkUp
	Aria2LinkDown
)

var (
	eventBus *bus.Bus
	eventMap = make(map[Type]string)
)

func init() {
	eventBus = bus.New()

	eventMap[ScaleWorker] = "event.scale.worker"
	eventMap[Download] = "event.download"
	eventMap[Aria2Test] = "event.aria2.test"
	eventMap[Aria2LinkUp] = "event.aria2.link_up"
	eventMap[Aria2LinkDown] = "event.aria2.link_down"
}

func GetBus() *bus.Bus {
	return eventBus
}

func (t Type) String() string {
	return eventMap[t]
}
