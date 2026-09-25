package script

import (
	"context"
	"testing"
	"time"
)

func TestExecutePureScript(t *testing.T) {
	result, err := Execute(context.Background(), Request{Script: "return { title: input.title.trim().toUpperCase() };", Input: map[string]any{"title": " hello "}})
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	values, ok := result.Output.(map[string]any)
	if !ok || values["title"] != "HELLO" {
		t.Fatalf("unexpected script result: %#v", result.Output)
	}
}

func TestExecuteTimeout(t *testing.T) {
	_, err := Execute(context.Background(), Request{Script: "while (true) {}", Timeout: 20 * time.Millisecond})
	if err == nil {
		t.Fatal("expected script timeout")
	}
}

func TestExecuteOutputLimit(t *testing.T) {
	_, err := Execute(context.Background(), Request{Script: "return '123456789';", MaxOutputBytes: 4})
	if err == nil {
		t.Fatal("expected output size error")
	}
}
