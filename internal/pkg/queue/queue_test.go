package queue

import (
	"testing"
	"time"
)

func TestQueuePushPop(t *testing.T) {
	q := NewQueue[int]("test", 2)
	q.Push(1)
	q.Push(2)
	if got, ok := q.Pop(); !ok || got != 1 {
		t.Fatalf("Pop() = %d, %v; want 1, true", got, ok)
	}
	if got, ok := q.Pop(); !ok || got != 2 {
		t.Fatalf("Pop() = %d, %v; want 2, true", got, ok)
	}
	if _, ok := q.Pop(); ok {
		t.Fatal("empty queue Pop() should return false")
	}
}

func TestQueuePopWaitTimeoutReceivesItem(t *testing.T) {
	q := NewQueue[int]("test", 1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		q.Push(42)
	}()
	if got, ok := q.PopWaitTimeout(time.Second); !ok || got != 42 {
		t.Fatalf("PopWaitTimeout() = %d, %v; want 42, true", got, ok)
	}
}

func TestQueuePopWaitTimeoutExpires(t *testing.T) {
	q := NewQueue[int]("test", 1)
	started := time.Now()
	if _, ok := q.PopWaitTimeout(30 * time.Millisecond); ok {
		t.Fatal("PopWaitTimeout() should time out")
	}
	if time.Since(started) < 25*time.Millisecond {
		t.Fatal("PopWaitTimeout() returned too early")
	}
}

func TestQueueBlockedPushUnblocksAfterPop(t *testing.T) {
	q := NewQueue[int]("test", 1)
	q.Push(1)
	done := make(chan struct{})
	go func() {
		q.Push(2)
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("Push() should block while queue is full")
	case <-time.After(20 * time.Millisecond):
	}
	q.Pop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Push() did not unblock after Pop()")
	}
}
