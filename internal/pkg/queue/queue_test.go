package queue

import (
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	for {
		<-time.After(1 * time.Second)
		t.Log("hello")
	}
}
