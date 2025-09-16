package bean

import (
	"context"
)

func DefaultValue[T any]() T {
	var defaultValue T
	return defaultValue
}

func ContextWithDefaultRegistry(ctx context.Context) context.Context {
	if RegistryFromContext(ctx) != nil {
		return ctx
	}
	registry := NewRegistry()
	newCtx := context.WithValue(ctx, DefaultValue[*Registry](), registry)
	lifecycleManager := NewLifecycleManager(newCtx)
	registry.SetLifecycleManager(lifecycleManager)
	return newCtx
}

func RegistryFromContext(ctx context.Context) Registry {
	registry := ctx.Value(DefaultValue[*Registry]())
	if registry == nil {
		return nil
	}
	return registry.(Registry)
}

func LifecycleFromContext(ctx context.Context) *LifecycleManager {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		panic("registry must be nil!")
	}
	return registry.(Registry).LifecycleManager()
}

func FromContext[T any](ctx context.Context) T {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return DefaultValue[T]()
	}
	service := registry.Get(DefaultValue[*T]())
	if service == nil {
		return DefaultValue[T]()
	}
	return service.(T)
}

func PtrFromContext[T any](ctx context.Context) *T {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return nil
	}
	servicePtr := registry.Get(DefaultValue[*T]())
	if servicePtr == nil {
		return nil
	}
	return servicePtr.(*T)
}

func MustRegister[T any](ctx context.Context, service T) {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		panic("missing service registry in context")
	}
	registry.Register(DefaultValue[*T](), service)
}

func MustRegisterPtr[T any](ctx context.Context, servicePtr *T) {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		panic("missing service registry in context")
	}
	registry.Register(DefaultValue[*T](), servicePtr)
}
