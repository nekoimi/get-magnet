package plugin

import (
	"context"
	"testing"
)

func TestWorkerSnapshotLifecycle(t *testing.T) {
	worker := NewWorker(NewRegistry())
	if snapshot := worker.Snapshot(); snapshot.Running {
		t.Fatal("new worker should not be running")
	}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot := worker.Snapshot()
	if !snapshot.Running || snapshot.WorkerCount != 1 || snapshot.StartedAt == nil {
		t.Fatalf("unexpected running snapshot: %#v", snapshot)
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot = worker.Snapshot()
	if snapshot.Running || snapshot.StoppedAt == nil {
		t.Fatalf("unexpected stopped snapshot: %#v", snapshot)
	}
}
