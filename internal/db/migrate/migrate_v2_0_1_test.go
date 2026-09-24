package migrate

import "testing"

func TestNormalizeCanonicalKey(t *testing.T) {
	if got := normalizeCanonicalKey("  ab c-01\t"); got != "ABC-01" {
		t.Fatalf("normalizeCanonicalKey() = %q", got)
	}
}

func TestBuildSourceURL(t *testing.T) {
	tests := []struct {
		host, path, want string
	}{
		{"javdb.com", "/v/abc", "https://javdb.com/v/abc"},
		{"https://javdb.com/", "v/abc", "https://javdb.com/v/abc"},
		{"", "/local", "/local"},
	}
	for _, tt := range tests {
		if got := buildSourceURL(tt.host, tt.path); got != tt.want {
			t.Errorf("buildSourceURL(%q, %q) = %q, want %q", tt.host, tt.path, got, tt.want)
		}
	}
}

func TestParseLegacyLinks(t *testing.T) {
	links, err := parseLegacyLinks(`[" magnet:?xt=1 ", "https://example.test/a", "magnet:?xt=1"]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 || links[0] != "magnet:?xt=1" || links[1] != "https://example.test/a" {
		t.Fatalf("unexpected links: %#v", links)
	}
}

func TestMigrationStatusInputs(t *testing.T) {
	if got := attributesJSON("actor"); got != `{"actress":"actor"}` {
		t.Fatalf("attributesJSON() = %s", got)
	}
	if got := linkType("magnet:?xt=urn:btih:abc"); got != "magnet" {
		t.Fatalf("linkType() = %q", got)
	}
	if !isValidResourceLink("https://example.test/a") || isValidResourceLink("not-a-link") {
		t.Fatal("resource link validation failed")
	}
}
