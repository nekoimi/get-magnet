package crawlpolicy

import "testing"

func TestBudgetValidationAndDomainMatching(t *testing.T) {
	b := Defaults("https://example.org/list")
	if err := b.Validate("https://example.org/list"); err != nil {
		t.Fatal(err)
	}
	if !b.Allows("https://example.org:8443/detail") || b.Allows("https://other.org/detail") || b.Allows("https://example.org.attacker.test/detail") || b.Allows("javascript:alert(1)") {
		t.Fatal("exact hostname allowlist failed")
	}
	for _, change := range []func(*Budget){func(b *Budget) { b.MaxTasks = 0 }, func(b *Budget) { b.MaxPages = 10001 }, func(b *Budget) { b.AllowedDomains = []string{"example.org", "example.org"} }, func(b *Budget) { b.AllowedDomains = []string{"*.example.org"} }} {
		invalid := b
		change(&invalid)
		if err := invalid.Validate("https://example.org/list"); err == nil {
			t.Fatalf("invalid budget accepted: %#v", invalid)
		}
	}
}
