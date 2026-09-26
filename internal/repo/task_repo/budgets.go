package task_repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nekoimi/scrapio/internal/crawlpolicy"
	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
	"xorm.io/xorm"
)

type BudgetExceeded struct{ Reason string }

func (e *BudgetExceeded) Error() string { return "run budget reached: " + e.Reason }

type LimitEvent struct {
	TaskID   int64  `json:"task_id"`
	PageRole string `json:"page_role"`
	URL      string `json:"url"`
	Reason   string `json:"reason"`
}

// InitializeRunBudget snapshots the published definition exactly once. Runs
// created before B02 are initialized lazily; restart never resets consumption.
func InitializeRunBudget(runID int64) (crawlpolicy.Budget, time.Time, error) {
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return crawlpolicy.Budget{}, time.Time{}, err
	}
	defer s.Rollback()
	if _, err := s.QueryString("SELECT id FROM workflow_runs WHERE id=? FOR NO KEY UPDATE", runID); err != nil {
		return crawlpolicy.Budget{}, time.Time{}, fmt.Errorf("lock run: %w", err)
	}
	var run table.WorkflowRun
	has, err := s.ID(runID).Get(&run)
	if err != nil || !has {
		return crawlpolicy.Budget{}, time.Time{}, fmt.Errorf("load run: %w", err)
	}
	var budget crawlpolicy.Budget
	if run.Budget != "" && run.Budget != "{}" {
		if err := json.Unmarshal([]byte(run.Budget), &budget); err != nil {
			return budget, time.Time{}, err
		}
	} else {
		var version table.WorkflowVersion
		if has, err := s.ID(run.WorkflowVersionId).Get(&version); err != nil || !has {
			return budget, time.Time{}, fmt.Errorf("load version: %w", err)
		}
		budget, _, err = crawlpolicy.FromDefinition(version.Definition)
		if err != nil {
			return budget, time.Time{}, err
		}
		encoded, _ := json.Marshal(budget)
		if _, err := s.Exec("UPDATE workflow_runs SET budget=CAST(? AS jsonb) WHERE id=?", string(encoded), runID); err != nil {
			return budget, time.Time{}, fmt.Errorf("write budget: %w", err)
		}
	}
	started := run.CreatedAt
	if err := s.Commit(); err != nil {
		return budget, time.Time{}, err
	}
	return budget, started.Add(time.Duration(budget.MaxDurationSeconds) * time.Second), nil
}

func limitEvent(s *xorm.Session, runID, taskID int64, role, pageURL, reason string) error {
	_, err := s.Exec(`INSERT INTO run_limit_events(run_id,task_id,page_role,url,reason) VALUES(?,?,?,?,?) ON CONFLICT DO NOTHING`, runID, taskID, role, pageURL, reason)
	return err
}

func RecordLimit(runID, taskID int64, role, pageURL, reason string) error {
	s := db.Instance().NewSession()
	defer s.Close()
	return limitEvent(s, runID, taskID, role, pageURL, reason)
}

func budgetFromRun(s *xorm.Session, runID int64) (crawlpolicy.Budget, time.Time, bool, error) {
	var run table.WorkflowRun
	has, err := s.ID(runID).Get(&run)
	if err != nil || !has {
		return crawlpolicy.Budget{}, time.Time{}, false, errors.New("run not found")
	}
	if run.Budget == "" || run.Budget == "{}" {
		return crawlpolicy.Budget{}, time.Time{}, false, nil
	}
	var budget crawlpolicy.Budget
	err = json.Unmarshal([]byte(run.Budget), &budget)
	started := run.CreatedAt
	return budget, started.Add(time.Duration(budget.MaxDurationSeconds) * time.Second), true, err
}

// childBudgetReason runs under the run lock, so concurrent parents cannot
// oversubscribe the task or per-page admission limits.
func childBudgetReason(s *xorm.Session, task *table.CrawlTask, parent *table.CrawlTask) (string, error) {
	b, deadline, enabled, err := budgetFromRun(s, task.RunId)
	if err != nil || !enabled {
		return "", err
	}
	if time.Now().After(deadline) {
		return "max_duration_seconds", nil
	}
	var input struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
		return "", err
	}
	if !b.Allows(input.URL) {
		return "allowed_domains", nil
	}
	task.Depth = parent.Depth + 1
	if task.Depth > b.MaxDepth {
		return "max_depth", nil
	}
	count, err := s.Where("run_id=?", task.RunId).Count(new(table.CrawlTask))
	if err != nil {
		return "", err
	}
	if count >= int64(b.MaxTasks) {
		return "max_tasks", nil
	}
	if task.StepName == "detail" {
		count, err = s.Where("run_id=? AND parent_task_id=? AND step_name='detail'", task.RunId, parent.Id).Count(new(table.CrawlTask))
		if err != nil {
			return "", err
		}
		if count >= int64(b.MaxDiscoveredPerPage) {
			return "max_discovered_per_page", nil
		}
	}
	return "", nil
}

// ReservePage persists one slot per task. Retried attempts reuse this slot.
func ReservePage(claim *Claim, pageURL string) error {
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	defer s.Rollback()
	rows, err := s.QueryString("SELECT status FROM workflow_runs WHERE id=? FOR NO KEY UPDATE", claim.Task.RunId)
	if err != nil {
		return err
	}
	if len(rows) == 0 || (rows[0]["status"] != RunQueued && rows[0]["status"] != RunRunning) {
		return ErrStaleAttempt
	}
	rows, err = s.QueryString("SELECT id FROM crawl_tasks WHERE id=? AND status=? AND lease_owner=? AND attempt_count=? AND lease_until>NOW() FOR NO KEY UPDATE", claim.Task.Id, TaskRunning, claim.Attempt.WorkerId, claim.Attempt.AttemptNo)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrStaleAttempt
	}
	b, deadline, enabled, err := budgetFromRun(s, claim.Task.RunId)
	if err != nil {
		return err
	}
	if !enabled {
		return errors.New("workflow budget was not initialized")
	}
	var task table.CrawlTask
	if _, err := s.ID(claim.Task.Id).Get(&task); err != nil {
		return err
	}
	reason := ""
	if time.Now().After(deadline) {
		reason = "max_duration_seconds"
	} else if !b.Allows(pageURL) {
		reason = "allowed_domains"
	} else if task.Depth > b.MaxDepth {
		reason = "max_depth"
	}
	if reason == "" && !task.PageReserved {
		count, err := s.Where("run_id=? AND page_reserved=true", task.RunId).Count(new(table.CrawlTask))
		if err != nil {
			return err
		}
		if count >= int64(b.MaxPages) {
			reason = "max_pages"
		} else {
			u, _ := url.Parse(pageURL)
			reason, err = reserveDomain(s, task.RunId, b, u.Hostname())
			if err != nil {
				return err
			}
			if reason == "" {
				if _, err := s.ID(task.Id).Cols("page_reserved", "page_host").Update(&table.CrawlTask{PageReserved: true, PageHost: u.Hostname()}); err != nil {
					return err
				}
			}
		}
	}
	if reason != "" {
		if err := limitEvent(s, task.RunId, task.Id, task.StepName, pageURL, reason); err != nil {
			return err
		}
	}
	if err := s.Commit(); err != nil {
		return err
	}
	if reason != "" {
		return &BudgetExceeded{reason}
	}
	return nil
}

func reserveDomain(s *xorm.Session, runID int64, b crawlpolicy.Budget, hostname string) (string, error) {
	rows, err := s.QueryString("SELECT hostname FROM run_domains WHERE run_id=?", runID)
	if err != nil {
		return "", err
	}
	for _, row := range rows {
		if row["hostname"] == hostname {
			return "", nil
		}
	}
	if len(rows) >= b.MaxDomains {
		return "max_domains", nil
	}
	_, err = s.Exec("INSERT INTO run_domains(run_id,hostname) VALUES(?,?) ON CONFLICT DO NOTHING", runID, hostname)
	return "", err
}

// CheckPageURL also runs for HTTP redirects before the redirected request.
func CheckPageURL(claim *Claim, pageURL string) error {
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	defer s.Rollback()
	rows, err := s.QueryString("SELECT status FROM workflow_runs WHERE id=? FOR NO KEY UPDATE", claim.Task.RunId)
	if err != nil {
		return err
	}
	if len(rows) == 0 || (rows[0]["status"] != RunQueued && rows[0]["status"] != RunRunning) {
		return ErrStaleAttempt
	}
	if err := CheckAttempt(claim.Task.Id, claim.Attempt); err != nil {
		return err
	}
	b, deadline, _, err := budgetFromRun(s, claim.Task.RunId)
	if err != nil {
		return err
	}
	reason := ""
	if time.Now().After(deadline) {
		reason = "max_duration_seconds"
	} else if !b.Allows(pageURL) {
		reason = "allowed_domains"
	} else {
		u, _ := url.Parse(pageURL)
		reason, err = reserveDomain(s, claim.Task.RunId, b, u.Hostname())
		if err != nil {
			return err
		}
	}
	if reason != "" {
		if err := limitEvent(s, claim.Task.RunId, claim.Task.Id, claim.Task.StepName, pageURL, reason); err != nil {
			return err
		}
	}
	if err := s.Commit(); err != nil {
		return err
	}
	if reason != "" {
		return &BudgetExceeded{reason}
	}
	return nil
}

func CompleteLimited(claim *Claim, pageURL, reason string) error {
	if err := CheckAttempt(claim.Task.Id, claim.Attempt); err != nil {
		return err
	}
	if err := RecordLimit(claim.Task.RunId, claim.Task.Id, claim.Task.StepName, pageURL, reason); err != nil {
		return err
	}
	output, _ := json.Marshal(map[string]any{"limited": true, "stop_reason": reason, "url": pageURL})
	return completeStatus(claim.Task.Id, claim.Attempt.Id, string(output), TaskLimited)
}

func LimitEvents(runID int64, size int) ([]LimitEvent, error) {
	if size < 1 || size > 100 {
		size = 100
	}
	rows, err := db.Instance().QueryString("SELECT task_id,page_role,url,reason FROM run_limit_events WHERE run_id=? ORDER BY id LIMIT ?", runID, size)
	result := make([]LimitEvent, 0, len(rows))
	for _, row := range rows {
		var id int64
		_, _ = fmt.Sscan(row["task_id"], &id)
		result = append(result, LimitEvent{id, row["page_role"], row["url"], row["reason"]})
	}
	return result, err
}

// ExpireBudgetRuns fences every remaining task, including delayed retries.
// This also finalizes an expired run when no worker is available to claim it.
func ExpireBudgetRuns() error {
	rows, err := db.Instance().QueryString(`SELECT id FROM workflow_runs WHERE status IN ('queued','running') AND budget <> '{}'::jsonb AND created_at + (budget->>'max_duration_seconds')::int * INTERVAL '1 second' <= NOW()`)
	if err != nil {
		return err
	}
	for _, row := range rows {
		var runID int64
		_, _ = fmt.Sscan(row["id"], &runID)
		if err := expireBudgetRun(runID); err != nil {
			return err
		}
	}
	return nil
}

func expireBudgetRun(runID int64) error {
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	defer s.Rollback()
	rows, err := s.QueryString("SELECT status FROM workflow_runs WHERE id=? FOR NO KEY UPDATE", runID)
	if err != nil {
		return err
	}
	if len(rows) == 0 || (rows[0]["status"] != RunQueued && rows[0]["status"] != RunRunning) {
		return nil
	}
	_, deadline, enabled, err := budgetFromRun(s, runID)
	if err != nil {
		return err
	}
	if !enabled || time.Now().Before(deadline) {
		return nil
	}
	var tasks []table.CrawlTask
	if err := s.Where("run_id=? AND status IN (?,?)", runID, TaskQueued, TaskRunning).Find(&tasks); err != nil {
		return err
	}
	now := time.Now()
	for _, task := range tasks {
		if err := limitEvent(s, runID, task.Id, task.StepName, taskURL(task), "max_duration_seconds"); err != nil {
			return err
		}
	}
	if _, err := s.Where("run_id=? AND status IN (?,?)", runID, TaskQueued, TaskRunning).Cols("status", "lease_owner", "lease_until", "finished_at", "updated_at", "error_message").Update(&table.CrawlTask{Status: TaskLimited, LeaseOwner: "", LeaseUntil: nil, FinishedAt: &now, UpdatedAt: now, ErrorMessage: "run budget reached: max_duration_seconds"}); err != nil {
		return err
	}
	if _, err := s.Where("task_id IN (SELECT id FROM crawl_tasks WHERE run_id=?) AND status=?", runID, TaskRunning).Cols("status", "finished_at", "error_message").Update(&table.TaskAttempt{Status: TaskLimited, FinishedAt: &now, ErrorMessage: "run duration budget expired"}); err != nil {
		return err
	}
	if err := s.Commit(); err != nil {
		return err
	}
	if len(tasks) > 0 {
		return tryFinishRun(tasks[0].Id)
	}
	return nil
}

func taskURL(task table.CrawlTask) string {
	var input struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal([]byte(task.Input), &input)
	return input.URL
}

type Coverage struct {
	Tasks              int64            `json:"tasks"`
	PagesReserved      int64            `json:"pages_reserved"`
	PagesWithDocuments int64            `json:"pages_with_documents"`
	MaxDepth           int64            `json:"max_depth"`
	Domains            []string         `json:"domains"`
	LimitedTasks       int64            `json:"limited_tasks"`
	LimitReasons       map[string]int64 `json:"limit_reasons"`
}

func coverage(s *xorm.Session, runID int64) (Coverage, error) {
	c := Coverage{Domains: []string{}, LimitReasons: map[string]int64{}}
	rows, err := s.QueryString(`SELECT count(*) AS tasks,count(*) FILTER(WHERE page_reserved) AS pages_reserved,count(*) FILTER(WHERE output_document_id IS NOT NULL) AS pages_with_documents,COALESCE(max(depth),0) AS max_depth,count(*) FILTER(WHERE status='limited') AS limited_tasks FROM crawl_tasks WHERE run_id=?`, runID)
	if err != nil {
		return c, err
	}
	if len(rows) > 0 {
		r := rows[0]
		_, _ = fmt.Sscan(r["tasks"], &c.Tasks)
		_, _ = fmt.Sscan(r["pages_reserved"], &c.PagesReserved)
		_, _ = fmt.Sscan(r["pages_with_documents"], &c.PagesWithDocuments)
		_, _ = fmt.Sscan(r["max_depth"], &c.MaxDepth)
		_, _ = fmt.Sscan(r["limited_tasks"], &c.LimitedTasks)
	}
	rows, err = s.QueryString("SELECT hostname FROM run_domains WHERE run_id=? ORDER BY hostname", runID)
	if err != nil {
		return c, err
	}
	for _, row := range rows {
		c.Domains = append(c.Domains, row["hostname"])
	}
	rows, err = s.QueryString("SELECT reason,count(*) AS count FROM run_limit_events WHERE run_id=? GROUP BY reason", runID)
	for _, row := range rows {
		var count int64
		_, _ = fmt.Sscan(row["count"], &count)
		c.LimitReasons[row["reason"]] = count
	}
	return c, err
}

func RunCoverage(runID int64) (Coverage, error) {
	s := db.Instance().NewSession()
	defer s.Close()
	return coverage(s, runID)
}
