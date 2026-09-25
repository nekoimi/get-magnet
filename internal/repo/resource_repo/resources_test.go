package resource_repo

import "testing"

func TestMergeAttributeMap(t *testing.T) {
	target := map[string]any{
		"title":    "demo",
		"delivery": map[string]any{"provider": "cloud-driver", "status": "queued"},
	}
	mergeAttributeMap(target, map[string]any{
		"delivery": map[string]any{"status": "completed", "files": []any{"movie.mp4"}},
		"new":      true,
	})
	delivery, ok := target["delivery"].(map[string]any)
	if !ok || delivery["provider"] != "cloud-driver" || delivery["status"] != "completed" {
		t.Fatalf("nested attributes were not merged: %#v", target)
	}
	if target["new"] != true {
		t.Fatalf("new attribute was not added: %#v", target)
	}
}
