package record

import (
	"reflect"
	"strings"
	"testing"
)

func TestPrepareValidatesCandidateAndKey(t *testing.T) {
	schema := Schema{UniqueKeyFields: []string{"url"}, EmptyValuePolicy: "preserve", Fields: []Field{{Key: "url", Type: "url", Required: true}, {Key: "title", Type: "string"}}}
	prepared, err := Prepare(schema, map[string]any{"url": " https://example.test/item/7 ", "title": " Example "})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Key != "https://example.test/item/7" || prepared.Values["title"] != "Example" || len(prepared.Hash) != 64 {
		t.Fatalf("unexpected prepared candidate: %+v", prepared)
	}
	cases := []map[string]any{{"title": "missing key"}, {"url": "file:///tmp/a"}, {"url": "https://example.test", "unknown": true}}
	for _, candidate := range cases {
		if _, err := Prepare(schema, candidate); err == nil {
			t.Fatalf("expected rejection: %#v", candidate)
		}
	}
}

func TestCompositeKeyCannotCollide(t *testing.T) {
	schema := Schema{UniqueKeyFields: []string{"a", "b"}, Fields: []Field{{Key: "a", Type: "string", Required: true}, {Key: "b", Type: "string", Required: true}}}
	first, _ := Prepare(schema, map[string]any{"a": "x\x1fy", "b": "z"})
	second, _ := Prepare(schema, map[string]any{"a": "x", "b": "y\x1fz"})
	if first.Key == second.Key {
		t.Fatal("composite keys collided")
	}
	if !strings.HasPrefix(first.Key, "[") {
		t.Fatalf("composite key should encode a tuple: %s", first.Key)
	}
}

func TestMergePreservesExistingAndUnionsMultiValue(t *testing.T) {
	old := map[string]any{"title": "Known", "links": []any{"magnet:?xt=urn:btih:AA"}}
	next := map[string]any{"title": "", "links": []any{"magnet:?xt=urn:btih:AA", "magnet:?xt=urn:btih:BB"}}
	merged, changed := Merge(old, next, "preserve")
	if merged["title"] != "Known" || !reflect.DeepEqual(changed, []string{"links"}) || len(merged["links"].([]any)) != 2 {
		t.Fatalf("merge: %#v, %#v", merged, changed)
	}
	if old["title"] != "Known" || len(old["links"].([]any)) != 1 {
		t.Fatal("merge mutated old value")
	}
}

func TestMergeEmptyValuePolicies(t *testing.T) {
	old := map[string]any{"title": "Known"}
	preserved, changed := Merge(old, map[string]any{"title": nil}, "preserve")
	if preserved["title"] != "Known" || len(changed) != 0 {
		t.Fatalf("preserve: %#v %#v", preserved, changed)
	}
	overwrote, changed := Merge(old, map[string]any{"title": nil}, "overwrite")
	if overwrote["title"] != nil || !reflect.DeepEqual(changed, []string{"title"}) {
		t.Fatalf("overwrite: %#v %#v", overwrote, changed)
	}
}

func TestLegacyDuplicateKey(t *testing.T) {
	if LegacyMagnetKey("AB-123#duplicate:42") != "AB-123" {
		t.Fatal("duplicate suffix not removed")
	}
	if LegacyMagnetKey("https://example.test/a") != "https://example.test/a" {
		t.Fatal("URL key changed")
	}
}
