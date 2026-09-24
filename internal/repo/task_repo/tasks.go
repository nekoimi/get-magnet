package task_repo

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/repo/resource_repo"
)

const (
	RunQueued      = "queued"
	RunRunning     = "running"
	RunSucceeded   = "succeeded"
	RunFailed      = "failed"
	RunCancelled   = "cancelled"
	RunInterrupted = "interrupted"

	TaskQueued     = "queued"
	TaskRunning    = "running"
	TaskSucceeded  = "succeeded"
	TaskFailed     = "failed"
	TaskCancelled  = "cancelled"
	TaskDeadLetter = "dead_letter"
)

type TaskFilter struct {
	RunID  int64
	Status string
	Page   int
	Size   int
}

type RunFilter struct {
	Status string
	Page   int
	Size   int
}

type Claim struct {
	Task    table.CrawlTask
	Attempt table.TaskAttempt
}

func DatabaseUnavailable() bool { return db.Instance() == nil }

func CreateRun(sourceCode, sourceName, triggerType, input string) (*table.WorkflowRun, error) {
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	workflowID, versionID, err := ensureWorkflow(sourceCode, sourceName)
	if err != nil {
		return nil, err
	}
	if input == "" {
		input = "{}"
	}
	now := time.Now()
	run := &table.WorkflowRun{WorkflowId: workflowID, WorkflowVersionId: versionID, TriggerType: triggerType, Status: RunQueued, Input: input, Summary: "{}", CreatedAt: now}
	if _, err := db.Instance().InsertOne(run); err != nil {
		return nil, err
	}
	return run, nil
}

func CreateTask(runID, parentTaskID int64, stepName, taskType, input string, maxAttempts int) (*table.CrawlTask, error) {
	if runID <= 0 {
		return nil, errors.New("run id is required")
	}
	if input == "" {
		input = "{}"
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	task := &table.CrawlTask{RunId: runID, StepName: stepName, TaskType: taskType, Input: input, Status: TaskQueued, MaxAttempts: maxAttempts, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if parentTaskID > 0 {
		task.ParentTaskId = &parentTaskID
	}
	if _, err := db.Instance().InsertOne(task); err != nil {
		return nil, err
	}
	_, _ = db.Instance().ID(runID).Cols("status", "started_at").Update(&table.WorkflowRun{Status: RunRunning, StartedAt: ptrTime(time.Now())})
	return task, nil
}

// ClaimNext atomically claims one queued or expired task and creates its attempt.
func ClaimNext(workerID string, lease time.Duration) (*Claim, bool, error) {
	if db.Instance() == nil {
		return nil, false, errors.New("database is not initialized")
	}
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, false, err
	}
	var task table.CrawlTask
	has, err := s.Where("(status = ? AND (next_retry_at IS NULL OR next_retry_at <= NOW())) OR (status = ? AND lease_until < NOW())", TaskQueued, TaskRunning).Asc("created_at").Limit(1).Get(&task)
	if err != nil || !has {
		_ = s.Rollback()
		return nil, false, err
	}
	until := time.Now().Add(lease)
	result, err := s.ID(task.Id).Cols("status", "attempt_count", "lease_owner", "lease_until", "updated_at").Where("(status = ? AND (next_retry_at IS NULL OR next_retry_at <= NOW())) OR (status = ? AND lease_until < NOW())", TaskQueued, TaskRunning).Update(&table.CrawlTask{Status: TaskRunning, AttemptCount: task.AttemptCount + 1, LeaseOwner: workerID, LeaseUntil: &until, UpdatedAt: time.Now()})
	if err != nil || result == 0 {
		_ = s.Rollback()
		return nil, false, err
	}
	attempt := &table.TaskAttempt{TaskId: task.Id, AttemptNo: task.AttemptCount + 1, WorkerId: workerID, Status: TaskRunning, StartedAt: ptrTime(time.Now()), RequestSnapshot: task.Input, ResponseSnapshot: "{}"}
	if _, err := s.Insert(attempt); err != nil {
		_ = s.Rollback()
		return nil, false, err
	}
	if err := s.Commit(); err != nil {
		return nil, false, err
	}
	task.Status = TaskRunning
	task.AttemptCount++
	task.LeaseOwner = workerID
	task.LeaseUntil = &until
	return &Claim{Task: task, Attempt: *attempt}, true, nil
}

func StartAttempt(taskID int64, workerID string, input string) (*table.TaskAttempt, error) {
	if taskID <= 0 {
		return nil, errors.New("task id is required")
	}
	attemptNo, err := nextAttempt(taskID)
	if err != nil {
		return nil, err
	}
	leaseUntil := time.Now().Add(5 * time.Minute)
	claimed, err := db.Instance().ID(taskID).Cols("status", "attempt_count", "lease_owner", "lease_until", "updated_at").Where("(status = ? AND (next_retry_at IS NULL OR next_retry_at <= NOW())) OR (status = ? AND lease_until < NOW())", TaskQueued, TaskRunning).Update(&table.CrawlTask{Status: TaskRunning, AttemptCount: attemptNo, LeaseOwner: workerID, LeaseUntil: &leaseUntil, UpdatedAt: time.Now()})
	if err != nil {
		return nil, err
	}
	if claimed == 0 {
		return nil, errors.New("task is already claimed or unavailable")
	}
	attempt := &table.TaskAttempt{TaskId: taskID, AttemptNo: attemptNo, WorkerId: workerID, Status: TaskRunning, StartedAt: ptrTime(time.Now()), RequestSnapshot: fallbackJSON(input), ResponseSnapshot: "{}"}
	if _, err := db.Instance().InsertOne(attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

func Complete(taskID, attemptID int64, response string) error {
	now := time.Now()
	attempt := new(table.TaskAttempt)
	if has, err := db.Instance().ID(attemptID).Get(attempt); err != nil || !has {
		if err != nil {
			return err
		}
		return errors.New("task attempt not found")
	}
	duration := now.Sub(timeOrZero(attempt.StartedAt)).Milliseconds()
	if _, err := db.Instance().ID(attemptID).Cols("status", "finished_at", "duration_ms", "response_snapshot").Update(&table.TaskAttempt{Status: TaskSucceeded, FinishedAt: &now, DurationMs: duration, ResponseSnapshot: fallbackJSON(response)}); err != nil {
		return err
	}
	if _, err := db.Instance().ID(taskID).Cols("status", "lease_owner", "lease_until", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskSucceeded, LeaseOwner: "", UpdatedAt: now, FinishedAt: &now}); err != nil {
		return err
	}
	return tryFinishRun(taskID, RunSucceeded)
}

func Fail(taskID, attemptID int64, cause error, retryable bool) (bool, error) {
	if cause == nil {
		cause = errors.New("task failed")
	}
	task := new(table.CrawlTask)
	has, err := db.Instance().ID(taskID).Get(task)
	if err != nil || !has {
		return false, err
	}
	now := time.Now()
	message := cause.Error()
	_, err = db.Instance().ID(attemptID).Cols("status", "finished_at", "error_message").Update(&table.TaskAttempt{Status: TaskFailed, FinishedAt: &now, ErrorMessage: message})
	if err != nil {
		return false, err
	}
	if retryable && task.AttemptCount < task.MaxAttempts {
		delay := backoff(task.AttemptCount)
		next := now.Add(delay)
		_, err = db.Instance().ID(taskID).Cols("status", "next_retry_at", "lease_owner", "lease_until", "error_message", "updated_at").Update(&table.CrawlTask{Status: TaskQueued, NextRetryAt: &next, LeaseOwner: "", ErrorMessage: message, UpdatedAt: now})
		return err == nil, err
	}
	_, err = db.Instance().ID(taskID).Cols("status", "lease_owner", "lease_until", "error_message", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskDeadLetter, LeaseOwner: "", ErrorMessage: message, UpdatedAt: now, FinishedAt: &now})
	if err != nil {
		return false, err
	}
	return false, tryFinishRun(taskID, RunFailed)
}

func RecoverExpiredLeases() (int64, error) {
	if db.Instance() == nil {
		return 0, errors.New("database is not initialized")
	}
	result, err := db.Instance().Where("status = ? AND lease_until < NOW()", TaskRunning).Cols("status", "next_retry_at", "lease_owner", "lease_until", "updated_at").Update(&table.CrawlTask{Status: TaskQueued, NextRetryAt: ptrTime(time.Now()), LeaseOwner: "", UpdatedAt: time.Now()})
	return result, err
}

func CancelTask(id int64) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	_, err := db.Instance().ID(id).Cols("status", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskCancelled, UpdatedAt: time.Now(), FinishedAt: ptrTime(time.Now())})
	return err
}

func CancelRun(id int64) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	if _, err := s.ID(id).Cols("status", "finished_at").Update(&table.WorkflowRun{Status: RunCancelled, FinishedAt: ptrTime(time.Now())}); err != nil {
		_ = s.Rollback()
		return err
	}
	if _, err := s.Where("run_id = ? AND status IN (?, ?)", id, TaskQueued, TaskRunning).Cols("status", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskCancelled, UpdatedAt: time.Now(), FinishedAt: ptrTime(time.Now())}); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}

func RetryTask(id int64) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	_, err := db.Instance().ID(id).Cols("status", "next_retry_at", "finished_at", "error_message", "updated_at").Update(&table.CrawlTask{Status: TaskQueued, NextRetryAt: ptrTime(time.Now()), FinishedAt: nil, ErrorMessage: "", UpdatedAt: time.Now()})
	return err
}

func ListRuns(filter RunFilter) ([]table.WorkflowRun, int64, error) {
	if db.Instance() == nil {
		return nil, 0, errors.New("database is not initialized")
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if filter.Status != "" {
		s.Where("status = ?", filter.Status)
	}
	total, err := s.Count(new(table.WorkflowRun))
	if err != nil {
		return nil, 0, err
	}
	rows := make([]table.WorkflowRun, 0)
	s = db.Instance().NewSession()
	defer s.Close()
	if filter.Status != "" {
		s.Where("status = ?", filter.Status)
	}
	err = s.Desc("created_at").Limit(filter.Size, (filter.Page-1)*filter.Size).Find(&rows)
	return rows, total, err
}

func ListTasks(filter TaskFilter) ([]table.CrawlTask, int64, error) {
	if db.Instance() == nil {
		return nil, 0, errors.New("database is not initialized")
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if filter.RunID > 0 {
		s.Where("run_id = ?", filter.RunID)
	}
	if filter.Status != "" {
		s.And("status = ?", filter.Status)
	}
	total, err := s.Count(new(table.CrawlTask))
	if err != nil {
		return nil, 0, err
	}
	rows := make([]table.CrawlTask, 0)
	s = db.Instance().NewSession()
	defer s.Close()
	if filter.RunID > 0 {
		s.Where("run_id = ?", filter.RunID)
	}
	if filter.Status != "" {
		s.And("status = ?", filter.Status)
	}
	err = s.Desc("created_at").Limit(filter.Size, (filter.Page-1)*filter.Size).Find(&rows)
	return rows, total, err
}

func ListAttempts(taskID int64, limit int) ([]table.TaskAttempt, error) {
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	rows := make([]table.TaskAttempt, 0)
	err := db.Instance().Where("task_id = ?", taskID).Desc("attempt_no").Limit(limit).Find(&rows)
	return rows, err
}

func ensureWorkflow(sourceCode, sourceName string) (int64, int64, error) {
	if strings.TrimSpace(sourceCode) == "" {
		sourceCode = "legacy"
	}
	sourceID, err := resource_repo.ResolveSourceID(nil, sourceCode, sourceName)
	if err != nil {
		return 0, 0, err
	}
	workflow := new(table.Workflow)
	has, err := db.Instance().Where("source_id = ? AND code = ?", sourceID, "default").Get(workflow)
	if err != nil {
		return 0, 0, err
	}
	if !has {
		workflow = &table.Workflow{SourceId: sourceID, Code: "default", Name: sourceName + " default", ResourceType: "magnet", Enabled: true}
		if _, err = db.Instance().InsertOne(workflow); err != nil {
			return 0, 0, err
		}
	}
	version := new(table.WorkflowVersion)
	has, err = db.Instance().Where("workflow_id = ? AND version = ?", workflow.Id, 1).Get(version)
	if err != nil {
		return 0, 0, err
	}
	if !has {
		version = &table.WorkflowVersion{WorkflowId: workflow.Id, Version: 1, Status: "published", Definition: "{}", CreatedAt: time.Now(), PublishedAt: ptrTime(time.Now())}
		if _, err = db.Instance().InsertOne(version); err != nil {
			return 0, 0, err
		}
		_, err = db.Instance().ID(workflow.Id).Cols("published_version_id").Update(&table.Workflow{PublishedVersionId: &version.Id})
		if err != nil {
			return 0, 0, err
		}
	}
	return workflow.Id, version.Id, nil
}

func nextAttempt(taskID int64) (int, error) {
	var item struct {
		Max int `xorm:"max"`
	}
	_, err := db.Instance().Table(new(table.TaskAttempt)).Select("COALESCE(MAX(attempt_no), 0) AS max").Where("task_id = ?", taskID).Get(&item)
	return item.Max + 1, err
}
func tryFinishRun(taskID int64, status string) error {
	var task table.CrawlTask
	has, err := db.Instance().ID(taskID).Get(&task)
	if err != nil || !has {
		return err
	}
	count, err := db.Instance().Where("run_id = ? AND status IN (?, ?, ?)", task.RunId, TaskQueued, TaskRunning, TaskFailed).Count(new(table.CrawlTask))
	if err != nil || count > 0 {
		return err
	}
	_, err = db.Instance().ID(task.RunId).Cols("status", "finished_at").Update(&table.WorkflowRun{Status: status, FinishedAt: ptrTime(time.Now())})
	return err
}
func backoff(attempt int) time.Duration {
	return BackoffDelay(attempt, rand.Int63n)
}
func ptrTime(t time.Time) *time.Time { return &t }
func timeOrZero(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
func fallbackJSON(value string) string {
	if strings.TrimSpace(value) == "" {
		return "{}"
	}
	var raw json.RawMessage
	if json.Unmarshal([]byte(value), &raw) != nil {
		b, _ := json.Marshal(map[string]string{"value": value})
		return string(b)
	}
	return value
}
func TaskInput(rawURL, origin string) string {
	b, _ := json.Marshal(map[string]string{"url": rawURL, "origin": origin})
	return string(b)
}
func TaskStep(rawURL string) string {
	if strings.Contains(strings.ToLower(rawURL), "page") {
		return "page"
	}
	return "crawl"
}

func BackoffDelay(attempt int, random func(int64) int64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 8 {
		attempt = 8
	}
	base := time.Second * time.Duration(1<<attempt)
	if random == nil {
		random = rand.Int63n
	}
	return base + time.Duration(random(int64(base/2)+1))
}
