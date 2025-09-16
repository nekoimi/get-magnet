package bean

import (
	"context"
)

// Lifecycle 生命周期接口
type Lifecycle interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type LifecycleWrapper struct {
	name      string
	startFunc func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
}

func NewLifecycle(name string, start func(ctx context.Context) error, stop func(ctx context.Context) error) Lifecycle {
	return &LifecycleWrapper{
		name:      name,
		startFunc: start,
		stopFunc:  stop,
	}
}

func (wrapper *LifecycleWrapper) Name() string {
	return wrapper.name
}

func (wrapper *LifecycleWrapper) Start(ctx context.Context) error {
	return wrapper.startFunc(ctx)
}

func (wrapper *LifecycleWrapper) Stop(ctx context.Context) error {
	return wrapper.stopFunc(ctx)
}
