package bus

import (
	"github.com/asaskevich/EventBus"
	log "github.com/sirupsen/logrus"
)

type Bus struct {
	eventBus EventBus.Bus
}

// 获取实例
func newEventBus() *Bus {
	return &Bus{
		eventBus: EventBus.New(),
	}
}

func (b *Bus) Publish(topic string, args ...interface{}) {
	b.eventBus.Publish(topic, args...)
}

func (b *Bus) Subscribe(topic string, fn interface{}) {
	err := b.eventBus.SubscribeAsync(topic, fn, true)
	if err != nil {
		log.Errorf("Event Subscribe error: %s", err.Error())
	}
}
