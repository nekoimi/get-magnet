package bean

import (
	log "github.com/sirupsen/logrus"
	"sync"
)

type Registry interface {
	LifecycleManager() *LifecycleManager
	SetLifecycleManager(lifecycleManager *LifecycleManager)
	Register(serviceType any, service any) any
	Get(serviceType any) any
}

func NewRegistry() Registry {
	return &defaultRegistry{
		serviceTypes:  make(map[any]any),
		serviceOrders: make([]any, 0),
	}
}

type defaultRegistry struct {
	lifecycleManager *LifecycleManager
	serviceTypes     map[any]any
	serviceOrders    []any
	access           sync.RWMutex
}

func (r *defaultRegistry) Register(serviceType any, service any) any {
	r.access.Lock()
	defer r.access.Unlock()
	oldService := r.serviceTypes[serviceType]
	r.serviceTypes[serviceType] = service
	log.Debugf("注册实例：val: %v", service)
	// 只有新注册的 key 才追加到顺序表
	if oldService == nil {
		r.serviceOrders = append(r.serviceOrders, service)
		if l, ok := service.(Lifecycle); ok {
			r.lifecycleManager.Register(l)
		}
	}
	return oldService
}

func (r *defaultRegistry) Get(serviceType any) any {
	r.access.RLock()
	defer r.access.RUnlock()
	service := r.serviceTypes[serviceType]
	log.Debugf("获取实例：key-%v ---> val: %v", serviceType, service)
	return service
}

func (r *defaultRegistry) LifecycleManager() *LifecycleManager {
	return r.lifecycleManager
}

func (r *defaultRegistry) SetLifecycleManager(lifecycleManager *LifecycleManager) {
	r.lifecycleManager = lifecycleManager
}
