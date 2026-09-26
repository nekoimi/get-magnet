package workflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nekoimi/scrapio/internal/drission_rod"
)

type fakeBrowser struct {
	called bool
	job    drission_rod.BrowserJob
}

func (f *fakeBrowser) Execute(_ context.Context, job drission_rod.BrowserJob) (drission_rod.BrowserResult, error) {
	f.called = true
	f.job = job
	return drission_rod.BrowserResult{FinalURL: job.URL + "?rendered=1", HTML: "<article>rendered</article>", ContentType: "text/html", ActionResults: []drission_rod.BrowserActionResult{{Index: 0, Type: "navigate", Success: true}}}, nil
}

func TestFetchHTTPHTMLJSONAndRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/page", http.StatusFound)
		case "/page":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<article>page</article>"))
		case "/items":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[1,2]}`))
		case "/missing":
			http.Error(w, "missing", http.StatusNotFound)
		default:
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	fetcher := Fetcher{}
	html, err := fetcher.Fetch(context.Background(), server.URL+"/redirect", FetchOptions{})
	if err != nil || html.Adapter != "http" || html.StatusCode != 200 || html.FinalURL != server.URL+"/page" || !strings.Contains(html.HTML, "article") || html.JSON != "" {
		t.Fatalf("HTML redirect: %+v %v", html, err)
	}
	items, err := fetcher.Fetch(context.Background(), server.URL+"/items", FetchOptions{})
	if err != nil || items.JSON != `{"items":[1,2]}` || items.HTML != "" {
		t.Fatalf("JSON fetch: %+v %v", items, err)
	}
	for _, tc := range []struct {
		path   string
		retry  bool
		status int
	}{{"/missing", false, 404}, {"/unavailable", true, 503}} {
		_, err = fetcher.Fetch(context.Background(), server.URL+tc.path, FetchOptions{})
		var fetchErr *FetchError
		if !errors.As(err, &fetchErr) || fetchErr.Stage != "response" || fetchErr.StatusCode != tc.status || fetchErr.Retryable != tc.retry {
			t.Fatalf("%s: %v", tc.path, err)
		}
	}
}

func TestFetchBrowserModeAndPublishValidation(t *testing.T) {
	browser := new(fakeBrowser)
	url := "https://example.test/page"
	result, err := (Fetcher{Browser: browser}).Fetch(context.Background(), url, FetchOptions{Mode: "browser", Actions: []FetchAction{{Type: "scroll", Value: "500"}}})
	if err != nil || !browser.called || len(browser.job.Actions) != 1 || result.Adapter != "browser" || result.FinalURL != url+"?rendered=1" || len(result.Actions) != 1 {
		t.Fatalf("browser fetch: %+v %v", result, err)
	}
	for _, options := range []FetchOptions{{Actions: []FetchAction{{Type: "wait", Selector: "#ready"}}}, {Mode: "browser", Actions: []FetchAction{{Type: "scroll", Value: "many"}}}, {Mode: "browser", Actions: []FetchAction{{Type: "script", Value: "alert(1)"}}}} {
		if err := validateFetchOptions(options); err == nil {
			t.Fatalf("unsupported fetch options accepted: %+v", options)
		}
	}
	if err := validateFetchOptions(FetchOptions{Mode: "browser", Actions: []FetchAction{{Type: "navigate", Value: url}, {Type: "wait", Selector: "#ready"}, {Type: "click", Selector: "#next"}, {Type: "input", Selector: "#q", Value: "test"}, {Type: "scroll", Value: "100"}}}); err != nil {
		t.Fatal(err)
	}
}
