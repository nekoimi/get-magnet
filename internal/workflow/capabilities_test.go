package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/nekoimi/scrapio/internal/drission_rod"
)

func TestPublishBoundaryRejectsUnexecutedCapabilities(t *testing.T) {
	base := `{"trigger":{"type":"manual","url":"https://example.test/detail"},"nodes":[{"name":"extract","type":"extract","config":{"fields":[{"name":"number","selector":"article","attribute":"data-code"}]}}]}`
	if _, err := ParseExecutableDefinition(base); err != nil {
		t.Fatalf("supported workflow rejected: %v", err)
	}
	cases := []struct{ name, raw, path string }{
		{"unhandled node", strings.Replace(base, `"type":"extract"`, `"type":"pagination"`, 1), "nodes[0].type"},
		{"unhandled trigger", strings.Replace(base, `"type":"manual"`, `"type":"cron"`, 1), "trigger.type"},
		{"ignored browser profile", strings.Replace(base, `"url":"https://example.test/detail"`, `"url":"https://example.test/detail","profile_id":"custom"`, 1), "trigger"},
		{"ignored node option", strings.Replace(base, `"fields":[`, `"recipe":"javdb","fields":[`, 1), "nodes[0].config.recipe"},
		{"unknown top-level option", strings.Replace(base, `"nodes":[`, `"future_option":true,"nodes":[`, 1), "definition"},
		{"unsupported JSONPath", strings.Replace(base, `"selector":"article"`, `"selector":"$.items[*]","selector_type":"jsonpath"`, 1), "nodes[0].config.fields[0].selector"},
		{"JSON with CSS", strings.Replace(base, `"fields":[`, `"content_type":"json","fields":[`, 1), "nodes[0].config.fields[0].selector_type"},
		{"unpersistable field", strings.Replace(base, `"name":"number"`, `"name":"title"`, 1), "nodes"},
		{"unpersistable after transform", `{"trigger":{"type":"manual","url":"https://example.test/detail"},"nodes":[{"name":"extract","type":"extract","config":{"fields":[{"name":"number","selector":"article"}]}},{"name":"drop","type":"transform","config":{"operations":[{"op":"delete","field":"number"}]}}]}`, "nodes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseExecutableDefinition(tc.raw)
			if err == nil || !strings.Contains(err.Error(), tc.path) {
				t.Fatalf("want blocker at %s, got %v", tc.path, err)
			}
		})
	}
}

func TestPublishRequiresDetailOutputForDiscovery(t *testing.T) {
	raw := `{"trigger":{"type":"manual","url":"https://example.test/list"},"nodes":[{"name":"links","type":"discover","config":{"fields":[{"name":"urls","selector":"a.detail","attribute":"href","multiple":true}]}},{"name":"number","type":"extract","config":{"run_on":"trigger","fields":[{"name":"number","selector":"article"}]}}]}`
	_, err := ParseExecutableDefinition(raw)
	if err == nil || !strings.Contains(err.Error(), "detail pages") {
		t.Fatalf("detail pages without a persistence key must be rejected: %v", err)
	}
}

func TestFixedDocumentContracts(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		b, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	// The browser and ordinary HTTP adapters both feed the same extractor.
	browser := drission_rod.BrowserResult{HTML: read("detail.html")}
	values, err := extractNodeValues(Node{Config: map[string]any{"fields": []any{
		map[string]any{"name": "number", "selector": "article", "attribute": "data-code"},
		map[string]any{"name": "title", "selector": "article h1"},
	}}}, browser)
	if err != nil || values["number"] != "AB-123" || values["title"] != "Example title" {
		t.Fatalf("detail extraction: %v, %#v", err, values)
	}
	links, err := Extract(ExtractRequest{Content: read("list.html"), Fields: []FieldRule{{Name: "urls", Selector: "a.detail", Attribute: "href", Multiple: true}}})
	if err != nil || len(links["urls"].([]any)) != 2 {
		t.Fatalf("list discovery: %v, %#v", err, links)
	}
	item, err := Extract(ExtractRequest{ContentType: "json", Content: read("items.json"), Fields: []FieldRule{{Name: "id", Selector: "$.items[0].id", Type: "integer", Required: true}}})
	if err != nil || item["id"] != int64(7) {
		t.Fatalf("JSON extraction: %v, %#v", err, item)
	}
}
