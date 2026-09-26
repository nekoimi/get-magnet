package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/nekoimi/scrapio/internal/drission_rod"
)

const maxFetchBytes = 10 << 20

type FetchResult struct {
	Adapter      string
	RequestID    string
	RequestedURL string
	FinalURL     string
	StatusCode   int
	ContentType  string
	HTML         string
	JSON         string
	Screenshot   []byte
	Duration     time.Duration
	Actions      []drission_rod.BrowserActionResult
}

type FetchError struct {
	Stage      string
	Code       string
	StatusCode int
	Retryable  bool
	Cause      error
	FinalURL   string
	Actions    []drission_rod.BrowserActionResult
}

func (e *FetchError) Error() string { return fmt.Sprintf("%s %s: %v", e.Stage, e.Code, e.Cause) }
func (e *FetchError) Unwrap() error { return e.Cause }
func (e *FetchError) FailureSnapshot() any {
	return map[string]any{"stage": e.Stage, "code": e.Code, "status_code": e.StatusCode, "retryable": e.Retryable, "final_url": e.FinalURL, "action_results": e.Actions}
}

type browserExecutor interface {
	Execute(context.Context, drission_rod.BrowserJob) (drission_rod.BrowserResult, error)
}

type Fetcher struct {
	HTTP     *http.Client
	Browser  browserExecutor
	AllowURL func(string) error
}

func (f Fetcher) Fetch(ctx context.Context, pageURL string, options FetchOptions) (FetchResult, error) {
	if f.AllowURL != nil {
		if err := f.AllowURL(pageURL); err != nil {
			return FetchResult{}, err
		}
	}
	mode := options.Mode
	if mode == "" {
		mode = "http"
	}
	if mode == "browser" {
		if f.Browser == nil {
			return FetchResult{}, &FetchError{Stage: "acquire", Code: "BROWSER_UNAVAILABLE", Retryable: true, Cause: errors.New("browser worker is unavailable")}
		}
		actions := make([]drission_rod.BrowserAction, 0, len(options.Actions))
		for _, action := range options.Actions {
			actions = append(actions, drission_rod.BrowserAction{Type: action.Type, Selector: action.Selector, Value: action.Value, Timeout: time.Duration(action.TimeoutMS) * time.Millisecond})
		}
		timeout := 60 * time.Second
		if options.TimeoutMS > 0 {
			timeout = time.Duration(options.TimeoutMS) * time.Millisecond
		}
		result, err := f.Browser.Execute(ctx, drission_rod.BrowserJob{URL: pageURL, Timeout: timeout, Actions: actions, Outputs: []string{"html", "json", "screenshot"}, ClosePage: true})
		if err != nil {
			var browserErr *drission_rod.BrowserError
			if errors.As(err, &browserErr) {
				return FetchResult{}, &FetchError{Stage: "browser", Code: browserErr.Code, Retryable: browserErr.Retryable, Cause: err, FinalURL: browserErr.FinalURL, Actions: browserErr.ActionResults}
			}
			return FetchResult{}, &FetchError{Stage: "browser", Code: "EXECUTION_FAILED", Cause: err}
		}
		finalURL := result.FinalURL
		if finalURL == "" {
			finalURL = pageURL
		}
		if f.AllowURL != nil {
			if err := f.AllowURL(finalURL); err != nil {
				return FetchResult{}, err
			}
		}
		return FetchResult{Adapter: "browser", RequestID: result.RequestID, RequestedURL: pageURL, FinalURL: finalURL, StatusCode: result.StatusCode, ContentType: result.ContentType, HTML: result.HTML, JSON: result.JSON, Screenshot: result.Screenshot, Duration: result.Duration, Actions: result.ActionResults}, nil
	}
	if mode != "http" {
		return FetchResult{}, &FetchError{Stage: "acquire", Code: "UNSUPPORTED_MODE", Cause: fmt.Errorf("mode %q", mode)}
	}
	client := f.HTTP
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if f.AllowURL != nil {
		copy := *client
		previous := client.CheckRedirect
		copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if err := f.AllowURL(req.URL.String()); err != nil {
				return err
			}
			if previous != nil {
				return previous(req, via)
			}
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		}
		client = &copy
	}
	if options.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.TimeoutMS)*time.Millisecond)
		defer cancel()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return FetchResult{}, &FetchError{Stage: "request", Code: "INVALID_URL", Cause: err}
	}
	request.Header.Set("Accept", "text/html,application/json,application/*+json;q=0.9")
	request.Header.Set("User-Agent", "scrapio/2.1")
	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return FetchResult{}, &FetchError{Stage: "request", Code: "NETWORK_ERROR", Retryable: true, Cause: err}
	}
	defer response.Body.Close()
	result := FetchResult{Adapter: "http", RequestedURL: pageURL, FinalURL: response.Request.URL.String(), StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), Duration: time.Since(started)}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return result, &FetchError{Stage: "response", Code: "HTTP_STATUS", StatusCode: response.StatusCode, Retryable: response.StatusCode == 408 || response.StatusCode == 429 || response.StatusCode >= 500, Cause: fmt.Errorf("HTTP %d", response.StatusCode), FinalURL: result.FinalURL}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxFetchBytes+1))
	if err != nil {
		return result, &FetchError{Stage: "response", Code: "READ_FAILED", Retryable: true, Cause: err}
	}
	if len(body) > maxFetchBytes {
		return result, &FetchError{Stage: "response", Code: "BODY_TOO_LARGE", Cause: fmt.Errorf("body exceeds %d bytes", maxFetchBytes)}
	}
	mediaType, _, err := mime.ParseMediaType(result.ContentType)
	if err != nil {
		mediaType = ""
	}
	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") || (mediaType == "" && json.Valid(body)) {
		if !json.Valid(body) {
			return result, &FetchError{Stage: "response", Code: "INVALID_JSON", Cause: errors.New("response is not valid JSON")}
		}
		result.JSON = string(body)
	} else if mediaType == "text/html" || mediaType == "application/xhtml+xml" || mediaType == "text/plain" || mediaType == "" {
		result.HTML = string(body)
	} else {
		return result, &FetchError{Stage: "response", Code: "UNSUPPORTED_CONTENT_TYPE", Cause: fmt.Errorf("content type %q", mediaType)}
	}
	result.Duration = time.Since(started)
	return result, nil
}
