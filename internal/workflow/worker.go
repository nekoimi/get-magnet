package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/drission_rod"
	"github.com/nekoimi/get-magnet/internal/repo/resource_repo"
	"github.com/nekoimi/get-magnet/internal/repo/task_repo"
	log "github.com/sirupsen/logrus"
)

const (
	workflowTaskType = "workflow"
	workflowLease    = 5 * time.Minute
	workflowPoll     = 500 * time.Millisecond
)

// Worker executes persisted workflow root tasks. It deliberately lives beside
// the DSL rather than in internal/crawler, keeping provider-specific workers
// and the generic workflow runtime independently deployable.
type Worker struct {
	browser *drission_rod.DrissionRod
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	count   int
}

func NewWorker() *Worker { return &Worker{} }

func (w *Worker) Name() string { return "WorkflowWorker" }

func (w *Worker) Start(parent context.Context) error {
	cfg := bean.PtrFromContext[config.Config](parent)
	w.browser = bean.PtrFromContext[drission_rod.DrissionRod](parent)
	w.count = 1
	if cfg != nil && cfg.Crawler != nil && cfg.Crawler.WorkerNum > 0 {
		w.count = cfg.Crawler.WorkerNum
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	for i := 0; i < w.count; i++ {
		w.wg.Add(1)
		go w.loop(ctx, i)
	}
	return nil
}

func (w *Worker) Stop(_ context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	return nil
}

func (w *Worker) loop(ctx context.Context, index int) {
	defer w.wg.Done()
	workerID := fmt.Sprintf("workflow-%d-%s", index, uuid.NewString()[:8])
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		claim, found, err := task_repo.ClaimNextType(workerID, workflowLease, workflowTaskType)
		if err != nil {
			if !task_repo.DatabaseUnavailable() {
				log.Warnf("领取 workflow 任务失败: %s", err)
			}
			if !waitWorkflow(ctx, workflowPoll) {
				return
			}
			continue
		}
		if !found {
			if !waitWorkflow(ctx, workflowPoll) {
				return
			}
			continue
		}
		if err := w.execute(ctx, claim); err != nil {
			retryable := false
			var browserErr *drission_rod.BrowserError
			if errors.As(err, &browserErr) {
				retryable = browserErr.Retryable
			}
			if _, failErr := task_repo.Fail(claim.Task.Id, claim.Attempt.Id, err, retryable); failErr != nil {
				log.Errorf("回写 workflow 失败状态异常: %s", failErr)
			}
			continue
		}
	}
}

func waitWorkflow(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *Worker) execute(ctx context.Context, claim *task_repo.Claim) error {
	if w.browser == nil {
		return &drission_rod.BrowserError{Code: "BROWSER_UNAVAILABLE", Retryable: true, Message: "browser worker is unavailable"}
	}
	run := new(table.WorkflowRun)
	has, err := db.Instance().ID(claim.Task.RunId).Get(run)
	if err != nil || !has {
		if err != nil {
			return err
		}
		return errors.New("workflow run not found")
	}
	version := new(table.WorkflowVersion)
	has, err = db.Instance().ID(run.WorkflowVersionId).Get(version)
	if err != nil || !has {
		if err != nil {
			return err
		}
		return errors.New("workflow version not found")
	}
	definition, err := ParseDefinition(version.Definition)
	if err != nil {
		return err
	}
	input := map[string]any{}
	if strings.TrimSpace(claim.Task.Input) != "" {
		if err := json.Unmarshal([]byte(claim.Task.Input), &input); err != nil {
			return fmt.Errorf("parse task input: %w", err)
		}
	}
	pageURL := stringValue(input["url"])
	if pageURL == "" {
		pageURL = definition.Trigger.URL
	}
	for _, node := range append(append([]Node{}, definition.Acquire...), definition.Nodes...) {
		if strings.EqualFold(node.Type, "navigate") && stringValue(node.Config["url"]) != "" {
			pageURL = stringValue(node.Config["url"])
			break
		}
	}
	if pageURL == "" {
		return errors.New("workflow entry url is required")
	}
	profile := definition.Trigger.ProfileID
	recipe := ""
	for _, node := range append(append([]Node{}, definition.Acquire...), definition.Nodes...) {
		if value := stringValue(node.Config["profile"]); value != "" {
			profile = value
		}
		if value := stringValue(node.Config["recipe"]); value != "" {
			recipe = value
		}
	}
	result, err := w.browser.Execute(ctx, drission_rod.BrowserJob{
		URL: pageURL, Profile: profile, Recipe: recipe, Timeout: 60 * time.Second,
		Outputs: []string{"html", "json", "screenshot"}, ClosePage: true,
	})
	if err != nil {
		return err
	}
	documentID, err := saveDocument(claim.Task.Id, pageURL, result)
	if err != nil {
		return err
	}
	if documentID > 0 {
		_, _ = db.Instance().ID(claim.Task.Id).Cols("output_document_id").Update(&table.CrawlTask{OutputDocumentId: &documentID})
	}

	values := map[string]any{}
	discoveredURLs := map[string]struct{}{}
	for _, node := range append(append([]Node{}, definition.Acquire...), definition.Nodes...) {
		if !nodeApplies(node, claim.Task.StepName) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(node.Type)) {
		case "extract":
			extracted, err := extractNodeValues(node, result)
			if err != nil {
				return fmt.Errorf("node %s: %w", node.Name, err)
			}
			for key, value := range extracted {
				values[key] = value
			}
		case "discover":
			if claim.Task.StepName != "trigger" {
				continue
			}
			extracted, err := extractNodeValues(node, result)
			if err != nil {
				return fmt.Errorf("node %s: %w", node.Name, err)
			}
			fieldName := stringValue(node.Config["url_field"])
			if fieldName == "" {
				fieldName = firstMapKey(extracted)
			}
			for _, rawURL := range stringSlice(extracted[fieldName]) {
				childURL, err := resolveURL(pageURL, rawURL)
				if err != nil || childURL == "" || childURL == pageURL {
					continue
				}
				if _, exists := discoveredURLs[childURL]; exists {
					continue
				}
				discoveredURLs[childURL] = struct{}{}
				if _, err := task_repo.CreateTask(run.Id, claim.Task.Id, "detail", workflowTaskType, task_repo.TaskInput(childURL, ""), 5); err != nil {
					return fmt.Errorf("create discovered task: %w", err)
				}
			}
		case "transform":
			if err := ApplyTransform(values, node.Config); err != nil {
				return fmt.Errorf("node %s: %w", node.Name, err)
			}
		case "validate":
			if err := ValidateValues(values, node.Config); err != nil {
				return fmt.Errorf("node %s: %w", node.Name, err)
			}
		}
	}
	resourceID, err := persistResource(values, pageURL, run.WorkflowId)
	if err != nil {
		return err
	}
	output := map[string]any{"document_id": documentID, "values": values}
	if resourceID > 0 {
		output["resource_id"] = resourceID
	} else {
		output["resource_persisted"] = false
	}
	encoded, _ := json.Marshal(output)
	_ = task_repo.UpdateRunSummary(run.Id, string(encoded))
	return task_repo.Complete(claim.Task.Id, claim.Attempt.Id, string(encoded))
}

func fieldRules(raw any) ([]FieldRule, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var fields []FieldRule
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("invalid extract.fields: %w", err)
	}
	return fields, nil
}

func extractNodeValues(node Node, result drission_rod.BrowserResult) (map[string]any, error) {
	fields, err := fieldRules(node.Config["fields"])
	if err != nil {
		return nil, err
	}
	content, contentType := result.HTML, "html"
	if strings.EqualFold(stringValue(node.Config["content_type"]), "json") || (content == "" && result.JSON != "") || (usesJSONPath(fields) && result.JSON != "") {
		content, contentType = result.JSON, "json"
	}
	if content == "" {
		return nil, errors.New("browser returned no extractable document")
	}
	return Extract(ExtractRequest{ContentType: contentType, Content: content, Fields: fields})
}

func firstMapKey(values map[string]any) string {
	for _, preferred := range []string{"urls", "url", "links", "link"} {
		if _, exists := values[preferred]; exists {
			return preferred
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		return key
	}
	return ""
}

func resolveURL(baseURL, rawURL string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	child, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || child.String() == "" {
		return "", err
	}
	resolved := base.ResolveReference(child)
	if !strings.EqualFold(resolved.Scheme, "http") && !strings.EqualFold(resolved.Scheme, "https") {
		return "", fmt.Errorf("unsupported discovered URL scheme: %s", resolved.Scheme)
	}
	if resolved.Host == "" {
		return "", errors.New("discovered URL host is empty")
	}
	return resolved.String(), nil
}

func usesJSONPath(fields []FieldRule) bool {
	for _, field := range fields {
		if strings.EqualFold(field.SelectorType, "jsonpath") || strings.EqualFold(field.SelectorType, "json_path") || strings.HasPrefix(strings.TrimSpace(field.Selector), "$") {
			return true
		}
	}
	return false
}

func nodeApplies(node Node, step string) bool {
	raw, ok := node.Config["run_on"]
	if !ok || raw == nil {
		return true
	}
	step = strings.ToLower(strings.TrimSpace(step))
	switch value := raw.(type) {
	case string:
		return strings.EqualFold(value, step)
	case []any:
		for _, item := range value {
			if strings.EqualFold(stringValue(item), step) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func saveDocument(taskID int64, pageURL string, result drission_rod.BrowserResult) (int64, error) {
	if db.Instance() == nil {
		return 0, errors.New("database is not initialized")
	}
	content, documentType := result.HTML, "html"
	if content == "" {
		content, documentType = result.JSON, "json"
	}
	if content == "" {
		return 0, errors.New("browser returned empty document")
	}
	hash := sha256.Sum256([]byte(content))
	metadata, _ := json.Marshal(map[string]any{"url": pageURL, "request_id": result.RequestID, "duration_ms": result.Duration.Milliseconds()})
	document := &table.Document{TaskId: &taskID, DocumentType: documentType, Content: content, ContentHash: hex.EncodeToString(hash[:]), ContentSize: int64(len(content)), Metadata: string(metadata), CreatedAt: time.Now()}
	if _, err := db.Instance().InsertOne(document); err != nil {
		return 0, err
	}
	if len(result.Screenshot) > 0 && len(result.Screenshot) <= 10*1024*1024 {
		assetHash := sha256.Sum256(result.Screenshot)
		_, err := db.Instance().Exec(`INSERT INTO document_assets (document_id, asset_type, content_type, content, content_hash, content_size, metadata) VALUES (?, 'screenshot', 'image/png', ?, ?, ?, '{}'::jsonb) ON CONFLICT (document_id, asset_type) DO UPDATE SET content = EXCLUDED.content, content_hash = EXCLUDED.content_hash, content_size = EXCLUDED.content_size`, document.Id, result.Screenshot, hex.EncodeToString(assetHash[:]), len(result.Screenshot))
		if err != nil {
			return document.Id, err
		}
	}
	return document.Id, nil
}

func persistResource(values map[string]any, pageURL string, workflowID int64) (int64, error) {
	title := stringValue(values["title"])
	number := stringValue(values["number"])
	if number == "" {
		number = stringValue(values["canonical_key"])
	}
	links := stringSlice(values["links"])
	optimal := stringValue(values["optimal_link"])
	if optimal == "" {
		optimal = stringValue(values["optimalLink"])
	}
	if number == "" && len(links) == 0 && optimal == "" {
		return 0, nil
	}
	workflow := new(table.Workflow)
	has, err := db.Instance().ID(workflowID).Get(workflow)
	if err != nil || !has {
		if err != nil {
			return 0, err
		}
		return 0, errors.New("workflow not found")
	}
	source, ok := resource_repo.GetSource(workflow.SourceId)
	if !ok {
		return 0, errors.New("workflow source not found")
	}
	parsed, err := url.Parse(pageURL)
	if err != nil || parsed == nil || parsed.Host == "" {
		return 0, errors.New("workflow entry url is invalid")
	}
	actress := stringValue(values["actress"])
	if actress == "" {
		actress = stringValue(values["actress0"])
	}
	resource, err := resource_repo.SaveCollected(source.Code, title, number, actress, parsed.Host, parsed.RequestURI(), links, optimal)
	if err != nil {
		return 0, err
	}
	return resource.Id, nil
}

func stringValue(value any) string {
	switch item := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(item)
	case json.Number:
		return item.String()
	case float64:
		return strconv.FormatFloat(item, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(item))
	}
}

func stringSlice(value any) []string {
	switch item := value.(type) {
	case []any:
		result := make([]string, 0, len(item))
		for _, value := range item {
			if text := stringValue(value); text != "" {
				result = append(result, text)
			}
		}
		return result
	case []string:
		return item
	case string:
		if strings.TrimSpace(item) == "" {
			return nil
		}
		return strings.Fields(item)
	default:
		return nil
	}
}
