package bus

import (
	"github.com/asaskevich/EventBus"
)

type Bus struct {
	eventBus EventBus.Bus
}

var eventBus *Bus

func init() {
	eventBus = &Bus{eventBus: EventBus.New()}
}

// Get 获取实例
func Get() *Bus {
	return eventBus
}

func (b *Bus) Publish(topic string, args ...interface{}) {
	b.eventBus.Publish(topic, args)
}

func (b *Bus) Subscribe(topic string, fn interface{}) {
	b.eventBus.SubscribeAsync(topic, fn, true)
}
