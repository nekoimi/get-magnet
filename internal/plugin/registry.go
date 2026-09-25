package plugin

import (
	"context"
	"errors"
	"sync"
)

type Task struct {
	ResourceID int64
	EventType  string
	Input      map[string]any
}

type Handler interface {
	Code() string
	Capabilities() []string
	Health(context.Context) error
	Handle(context.Context, Task) (output any, externalID string, err error)
}

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewRegistry() *Registry { return &Registry{handlers: map[string]Handler{}} }

func (r *Registry) Register(handler Handler) error {
	if handler == nil || handler.Code() == "" {
		return errors.New("plugin code is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[handler.Code()]; exists {
		return errors.New("plugin already registered")
	}
	r.handlers[handler.Code()] = handler
	return nil
}

func (r *Registry) Get(code string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[code]
	return handler, ok
}

func (r *Registry) List() []Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Handler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		result = append(result, handler)
	}
	return result
}
