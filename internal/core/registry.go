package core

import (
	"context"
	"sync"
)

type Registry interface {
	LifecycleManager() *LifecycleManager
	Register(serviceType any, service any) any
	Get(serviceType any) any
}

func NewRegistry(ctx context.Context) Registry {
	return &defaultRegistry{
		lifecycleManager: NewLifecycleManager(ctx),
		serviceTypes:     make(map[any]any),
	}
}

type defaultRegistry struct {
	lifecycleManager *LifecycleManager
	serviceTypes     map[any]any
	access           sync.RWMutex
}

func (r *defaultRegistry) Register(serviceType any, service any) any {
	r.access.Lock()
	defer r.access.Unlock()
	oldService := r.serviceTypes[serviceType]
	r.serviceTypes[serviceType] = service
	if l, ok := service.(Lifecycle); ok {
		r.lifecycleManager.Register(l)
	}
	return oldService
}

func (r *defaultRegistry) Get(serviceType any) any {
	r.access.RLock()
	defer r.access.RUnlock()
	return r.serviceTypes[serviceType]
}

func (r *defaultRegistry) LifecycleManager() *LifecycleManager {
	return r.lifecycleManager
}
