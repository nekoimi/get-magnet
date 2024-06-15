package test

import (
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	timer := time.NewTimer(1 * time.Second)

	<-timer.C

	t.Log("timer")

	for {
	}
}

func TestTicker(t *testing.T) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		t.Log("ticker")
	}
}
