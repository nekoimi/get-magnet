package plugin

import (
	"context"
	"testing"
)

type testHandler struct{}

func (testHandler) Code() string                 { return "demo" }
func (testHandler) Capabilities() []string       { return []string{"test"} }
func (testHandler) Health(context.Context) error { return nil }
func (testHandler) Handle(context.Context, Task) (any, string, error) {
	return map[string]any{"ok": true}, "", nil
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testHandler{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(testHandler{}); err == nil {
		t.Fatal("expected duplicate plugin error")
	}
	if _, ok := registry.Get("demo"); !ok {
		t.Fatal("plugin not found")
	}
}
