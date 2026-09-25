package workflow

import "testing"

func TestDefinitionValidation(t *testing.T) {
	definition, err := ParseDefinition(`{"trigger":{"type":"manual"},"nodes":[{"name":"extract","type":"extract","config":{"fields":[]}}]}`)
	if err != nil || definition.Trigger.Type != "manual" {
		t.Fatalf("unexpected definition: %#v, %v", definition, err)
	}
	if _, err := ParseDefinition(`{"trigger":{"type":"manual"},"nodes":[{"name":"x","type":"unknown"}]}`); err == nil {
		t.Fatal("expected unsupported node error")
	}
	if _, err := ParseDefinition(`{"trigger":{"type":"manual"},"nodes":[{"name":"x","type":"transform","config":{"operations":{}}}]}`); err == nil {
		t.Fatal("expected invalid transform operations error")
	}
	if _, err := ParseDefinition(`{"trigger":{"type":"manual"},"nodes":[{"name":"x","type":"extract","config":{"fields":[],"run_on":42}}]}`); err == nil {
		t.Fatal("expected invalid run_on error")
	}
	if _, err := ParseDefinition(`{"trigger":{"type":"manual"},"nodes":[{"name":"x","type":"script","config":{"script":"return {};"}}]}`); err != nil {
		t.Fatalf("expected script node to validate: %v", err)
	}
	if _, err := ParseDefinition(`{"trigger":{"type":"manual"},"nodes":[{"name":"x","type":"script","config":{}}]}`); err == nil {
		t.Fatal("expected empty script to fail validation")
	}
}

func TestDefinitionTriggerOptions(t *testing.T) {
	definition, err := ParseDefinition(`{"trigger":{"type":"api","url":"https://example.test","profile_id":"main","concurrency":2},"nodes":[{"name":"extract","type":"extract","config":{"fields":[]}}]}`)
	if err != nil {
		t.Fatalf("expected trigger options to validate: %v", err)
	}
	if definition.Trigger.ProfileID != "main" || definition.Trigger.Concurrency != 2 {
		t.Fatalf("trigger options were not decoded: %+v", definition.Trigger)
	}
	if _, err := ParseDefinition(`{"trigger":{"type":"manual","concurrency":-1},"nodes":[{"name":"extract","type":"extract","config":{"fields":[]}}]}`); err == nil {
		t.Fatal("expected negative concurrency to fail validation")
	}
}

func TestExtractCSSXPathAndJSONPath(t *testing.T) {
	htmlResult, err := Extract(ExtractRequest{Content: `<article><a href="magnet:?xt=1">Title 01</a></article>`, Fields: []FieldRule{{Name: "title", Selector: "article a"}, {Name: "link", Selector: "//article/a", SelectorType: "xpath", Attribute: "href"}}})
	if err != nil || htmlResult["title"] != "Title 01" || htmlResult["link"] != "magnet:?xt=1" {
		t.Fatalf("unexpected HTML extraction: %#v, %v", htmlResult, err)
	}
	jsonResult, err := Extract(ExtractRequest{ContentType: "json", Content: `{"items":[{"id":7}]}`, Fields: []FieldRule{{Name: "id", Selector: "$.items[0].id", Type: "integer", Required: true}}})
	if err != nil || jsonResult["id"] != int64(7) {
		t.Fatalf("unexpected JSON extraction: %#v, %v", jsonResult, err)
	}
}

func TestWorkflowWorkerFieldFormat(t *testing.T) {
	fields, err := fieldRules([]any{map[string]any{"name": "id", "selector": "$.id"}})
	if err != nil || len(fields) != 1 || !usesJSONPath(fields) {
		t.Fatalf("unexpected worker field rules: %#v, %v", fields, err)
	}
	if usesJSONPath([]FieldRule{{Name: "title", Selector: "article h1"}}) {
		t.Fatal("CSS selector should not select JSON output")
	}
}

func TestTransformAndValidate(t *testing.T) {
	values := map[string]any{"title": "  Hello ", "tags": "one, two"}
	err := ApplyTransform(values, map[string]any{"operations": []any{
		map[string]any{"op": "trim", "field": "title"},
		map[string]any{"op": "lower", "field": "title"},
		map[string]any{"op": "split", "field": "tags", "separator": ","},
		map[string]any{"op": "rename", "field": "title", "to": "name"},
	}})
	if err != nil || values["name"] != "hello" {
		t.Fatalf("unexpected transform result: %#v, %v", values, err)
	}
	if err := ValidateValues(values, map[string]any{"fields": []any{
		map[string]any{"name": "name", "required": true, "type": "string", "regex": `^hello$`},
		map[string]any{"name": "tags", "required": true, "type": "array"},
	}}); err != nil {
		t.Fatalf("expected validation to pass: %v", err)
	}
	if err := ValidateValues(map[string]any{}, map[string]any{"fields": []any{map[string]any{"name": "name", "required": true}}}); err == nil {
		t.Fatal("expected missing required field to fail validation")
	}
}

func TestExtractMultipleAndResolveURL(t *testing.T) {
	values, err := Extract(ExtractRequest{Content: `<a class="detail" href="/one">One</a><a class="detail" href="/two">Two</a>`, Fields: []FieldRule{{Name: "urls", Selector: "a.detail", Attribute: "href", Multiple: true}}})
	if err != nil {
		t.Fatalf("multiple extraction failed: %v", err)
	}
	urls, ok := values["urls"].([]any)
	if !ok || len(urls) != 2 || urls[0] != "/one" || urls[1] != "/two" {
		t.Fatalf("unexpected multiple extraction: %#v", values["urls"])
	}
	resolved, err := resolveURL("https://example.test/list", "/detail/1")
	if err != nil || resolved != "https://example.test/detail/1" {
		t.Fatalf("unexpected resolved URL: %s, %v", resolved, err)
	}
	if _, err := resolveURL("https://example.test/list", "javascript:alert(1)"); err == nil {
		t.Fatal("expected unsafe discovered URL scheme to fail")
	}
}
