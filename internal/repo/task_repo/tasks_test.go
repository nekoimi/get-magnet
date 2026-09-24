package task_repo

import (
	"testing"
	"time"
)

func TestTaskInputAndStep(t *testing.T) {
	input := TaskInput("https://example.test/list?page=1", "demo")
	if input == "" || TaskStep("https://example.test/page/1") != "page" || TaskStep("https://example.test/list") != "crawl" {
		t.Fatalf("unexpected task metadata: %q", input)
	}
}

func TestBackoffDelayIsExponentialWithJitter(t *testing.T) {
	first := BackoffDelay(1, func(int64) int64 { return 0 })
	second := BackoffDelay(2, func(int64) int64 { return 0 })
	if first != 2*time.Second || second != 4*time.Second {
		t.Fatalf("unexpected delays: %s, %s", first, second)
	}
	if got := BackoffDelay(99, func(max int64) int64 { return max - 1 }); got <= second {
		t.Fatalf("max retry delay should grow, got %s", got)
	}
}
