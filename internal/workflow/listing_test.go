package workflow

import (
	"strings"
	"testing"
)

func TestListingResolvesAndFiltersLinks(t *testing.T) {
	opts := ListingOptions{DetailSelector: "a.item", NextSelector: "a.next", MaxPages: 3, MaxEmptyPages: 2}
	doc := FetchResult{FinalURL: "https://example.org/list?page=1", HTML: `<a class="item" href="/detail/1#x">one</a><a class="item" href="/detail/1#y">duplicate</a><a class="item" href="https://other.org/detail/2">other</a><a class="item" href="javascript:alert(1)">script</a><a class="item" href="detail/3">three</a><a class="next" href="?page=2">next</a>`}
	result, err := ExtractListing(opts, doc, "https://example.org/list?page=1")
	if err != nil || len(result.Details) != 2 || result.Details[0] != "https://example.org/detail/1" || result.Details[1] != "https://example.org/detail/3" || result.NextURL != "https://example.org/list?page=2" {
		t.Fatalf("listing: %#v %v", result, err)
	}
	doc.HTML = `<a class="next" href="?page=1">repeat</a>`
	result, err = ExtractListing(opts, doc, "https://example.org/list?page=1")
	if err != nil || listingStop(opts, 1, 0, result.NextURL, true) != "repeated_next_page" {
		t.Fatalf("repeated next: %#v %v", result, err)
	}
}

func TestListingStopConditions(t *testing.T) {
	opts := ListingOptions{MaxPages: 3, MaxEmptyPages: 2}
	for _, tc := range []struct {
		page, empty int
		next        string
		repeated    bool
		want        string
	}{
		{1, 0, "", false, "no_next_page"},
		{1, 0, "https://example.org/list", true, "repeated_next_page"},
		{3, 0, "https://example.org/list?page=4", false, "max_pages"},
		{2, 2, "https://example.org/list?page=3", false, "consecutive_empty_pages"},
		{2, 1, "https://example.org/list?page=3", false, ""},
	} {
		if got := listingStop(opts, tc.page, tc.empty, tc.next, tc.repeated); got != tc.want {
			t.Fatalf("%#v: got %q", tc, got)
		}
	}
}

func TestListingDefinitionBounds(t *testing.T) {
	d := Templates()[3].Definition
	if err := d.ValidateExecutable(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(){
		func() { d.Listing.MaxPages = 101 },
		func() { d.Listing.MaxPages = 10; d.Listing.NextSelector = "[" },
	} {
		change()
		if err := d.ValidateExecutable(); err == nil || !strings.Contains(err.Error(), "listing") {
			t.Fatalf("invalid listing accepted: %#v %v", d.Listing, err)
		}
	}
}
