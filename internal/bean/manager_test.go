package bean

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDatabaseFailureStopsOtherLifecycles(t *testing.T) {
	migrationErr := errors.New("migration failed")
	started := false
	m := &LifecycleManager{ctx: context.Background(), lifecycles: []Lifecycle{
		NewLifecycle("DB", func(context.Context) error { return migrationErr }, func(context.Context) error { return nil }),
		NewLifecycle("Worker", func(context.Context) error { started = true; return nil }, func(context.Context) error { return nil }),
	}}
	err := m.StartAndServe()
	if !errors.Is(err, migrationErr) || !strings.Contains(err.Error(), "DB") {
		t.Fatalf("expected database startup error, got %v", err)
	}
	if started {
		t.Fatal("worker started before database migration completed")
	}
}
