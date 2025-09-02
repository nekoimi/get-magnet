package core

import "context"

func DefaultValue[T any]() T {
	var defaultValue T
	return defaultValue
}

func ContextWithRegistry(ctx context.Context, registry Registry) context.Context {
	return context.WithValue(ctx, DefaultValue[*Registry](), registry)
}

func ContextWithDefaultRegistry(ctx context.Context) context.Context {
	if RegistryFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, DefaultValue[*Registry](), NewRegistry(ctx))
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

func ContextWith[T any](ctx context.Context, service T) context.Context {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		registry = NewRegistry(ctx)
		ctx = ContextWithRegistry(ctx, registry)
	}
	registry.Register(DefaultValue[*T](), service)
	return ctx
}

func ContextWithPtr[T any](ctx context.Context, servicePtr *T) context.Context {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		registry = NewRegistry(ctx)
		ctx = ContextWithRegistry(ctx, registry)
	}
	registry.Register(DefaultValue[*T](), servicePtr)
	return ctx
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
