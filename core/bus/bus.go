package bus

import (
	"github.com/asaskevich/EventBus"
	"log"
)

type Bus struct {
	eventBus EventBus.Bus
}

// New 获取实例
func New() *Bus {
	return &Bus{
		eventBus: EventBus.New(),
	}
}

func (b *Bus) Publish(topic string, args ...interface{}) {
	b.eventBus.Publish(topic, args)
}

func (b *Bus) Subscribe(topic string, fn interface{}) {
	err := b.eventBus.SubscribeAsync(topic, fn, true)
	if err != nil {
		log.Printf("Event Subscribe error: %s\n", err.Error())
		return
	}
}
