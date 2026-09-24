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
