package workflow

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/andybalholm/cascadia"
)

type ListingResult struct {
	Details []string `json:"details"`
	NextURL string   `json:"next_url,omitempty"`
}

func validateListingSelector(selector string) error {
	if _, err := cascadia.Parse(strings.TrimSpace(selector)); err != nil {
		return fmt.Errorf("invalid CSS selector %q: %w", selector, err)
	}
	return nil
}

// ExtractListing resolves page links and keeps the crawl on the entry host.
func ExtractListing(options ListingOptions, document FetchResult, entryURL string) (ListingResult, error) {
	var result ListingResult
	baseURL := document.FinalURL
	if baseURL == "" {
		baseURL = entryURL
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return result, err
	}
	entry, err := url.Parse(entryURL)
	if err != nil {
		return result, err
	}
	values, err := Extract(ExtractRequest{ContentType: "html", Content: document.HTML, Fields: []FieldRule{{Name: "details", Selector: options.DetailSelector, Attribute: "href", Multiple: true}}})
	if err != nil {
		return result, err
	}
	seen := map[string]bool{}
	for _, raw := range stringSlice(values["details"]) {
		resolved, ok := listingURL(base, entry, raw)
		if ok && !seen[resolved] {
			seen[resolved] = true
			result.Details = append(result.Details, resolved)
		}
	}
	if options.NextSelector != "" {
		values, err = Extract(ExtractRequest{ContentType: "html", Content: document.HTML, Fields: []FieldRule{{Name: "next", Selector: options.NextSelector, Attribute: "href"}}})
		if err != nil {
			return result, err
		}
		result.NextURL, _ = listingURL(base, entry, stringValue(values["next"]))
	}
	return result, nil
}

func listingStop(options ListingOptions, pageIndex, emptyStreak int, next string, repeated bool) string {
	if next == "" {
		return "no_next_page"
	}
	if repeated {
		return "repeated_next_page"
	}
	if pageIndex >= options.MaxPages {
		return "max_pages"
	}
	if emptyStreak >= options.MaxEmptyPages {
		return "consecutive_empty_pages"
	}
	return ""
}

// FetchForRole makes preview acquisition use the same action filtering as tasks.
func (d Definition) FetchForRole(role string) FetchOptions {
	if d.Listing != nil && role == "trigger" {
		role = "list"
	}
	options := d.Trigger.Fetch
	options.Actions = nil
	for _, action := range d.Trigger.Fetch.Actions {
		if action.RunOn == "" || action.RunOn == role {
			options.Actions = append(options.Actions, action)
		}
	}
	return options
}

func listingNormalized(u *url.URL) string {
	copy := *u
	copy.Fragment = ""
	copy.Host = strings.ToLower(copy.Host)
	return copy.String()
}

func listingURL(base, entry *url.URL, raw string) (string, bool) {
	if strings.TrimSpace(raw) == "" {
		return "", false
	}
	reference, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	resolved := base.ResolveReference(reference)
	if (resolved.Scheme != "http" && resolved.Scheme != "https") || !strings.EqualFold(resolved.Hostname(), entry.Hostname()) || resolved.User != nil {
		return "", false
	}
	return listingNormalized(resolved), true
}
