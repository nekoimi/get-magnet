package task_repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/repo/resource_repo"
)

const (
	RunQueued      = "queued"
	RunRunning     = "running"
	RunSucceeded   = "succeeded"
	RunPartial     = "partial"
	RunLimited     = "limited"
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
	Status     string
	ProjectID  int64
	WorkflowID int64
	Page       int
	Size       int
}

type Claim struct {
	Task    table.CrawlTask
	Attempt table.TaskAttempt
}

var ErrStaleAttempt = errors.New("task attempt is no longer active")

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
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	// Lock the run so cancellation cannot commit between the status check and
	// insertion of a child task.
	rows, err := s.QueryString("SELECT status FROM workflow_runs WHERE id = ? FOR UPDATE", runID)
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if len(rows) == 0 || (rows[0]["status"] != RunQueued && rows[0]["status"] != RunRunning) {
		_ = s.Rollback()
		return nil, errors.New("workflow run is no longer active")
	}
	if parentTaskID > 0 {
		parentRows, err := s.QueryString("SELECT status FROM crawl_tasks WHERE id = ? AND run_id = ? FOR UPDATE", parentTaskID, runID)
		if err != nil {
			_ = s.Rollback()
			return nil, err
		}
		if len(parentRows) == 0 || parentRows[0]["status"] != TaskRunning {
			_ = s.Rollback()
			return nil, ErrStaleAttempt
		}
	}
	// The run lock serializes child creation across workers and retries.
	// Returning the existing task prevents a replayed parent from duplicating
	// detail work. Non-workflow task types keep their historical behavior.
	if parentTaskID > 0 && taskType == "workflow" {
		var existing table.CrawlTask
		has, err := s.Where("run_id = ? AND task_type = ? AND step_name = ? AND input = ?::jsonb", runID, taskType, stepName, input).Get(&existing)
		if err != nil {
			_ = s.Rollback()
			return nil, err
		}
		if has {
			_ = s.Rollback()
			return &existing, nil
		}
	}
	if _, err := s.InsertOne(task); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if _, err := s.ID(runID).Where("status = ?", RunQueued).Cols("status", "started_at").Update(&table.WorkflowRun{Status: RunRunning, StartedAt: ptrTime(time.Now())}); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := s.Commit(); err != nil {
		return nil, err
	}
	return task, nil
}

// ClaimNext atomically claims one queued or expired task and creates its attempt.
func ClaimNext(workerID string, lease time.Duration) (*Claim, bool, error) {
	return claimNext(workerID, lease, "")
}

// ClaimNextType atomically claims one task of the requested type. Workflow
// execution uses this typed variant so it cannot consume legacy browser tasks.
func ClaimNextType(workerID string, lease time.Duration, taskType string) (*Claim, bool, error) {
	return claimNext(workerID, lease, strings.TrimSpace(taskType))
}

func claimNext(workerID string, lease time.Duration, taskType string) (*Claim, bool, error) {
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
	condition := "((status = ? AND (next_retry_at IS NULL OR next_retry_at <= NOW())) OR (status = ? AND lease_until < NOW())) AND attempt_count < max_attempts AND EXISTS (SELECT 1 FROM workflow_runs WHERE workflow_runs.id = crawl_tasks.run_id AND workflow_runs.status IN (?, ?))"
	args := []any{TaskQueued, TaskRunning, RunQueued, RunRunning}
	if taskType != "" {
		condition = condition + " AND task_type = ?"
		args = append(args, taskType)
	}
	has, err := s.Where(condition, args...).Asc("created_at").Limit(1).Get(&task)
	if err != nil || !has {
		_ = s.Rollback()
		return nil, false, err
	}
	until := time.Now().Add(lease)
	result, err := s.ID(task.Id).Cols("status", "attempt_count", "lease_owner", "lease_until", "updated_at").Where(condition, args...).Update(&table.CrawlTask{Status: TaskRunning, AttemptCount: task.AttemptCount + 1, LeaseOwner: workerID, LeaseUntil: &until, UpdatedAt: time.Now()})
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
	// A run becomes running when its first durable task is claimed. This keeps
	// queued runs meaningful even when execution is performed asynchronously.
	_, _ = db.Instance().ID(task.RunId).Cols("status", "started_at").Where("status = ?", RunQueued).Update(&table.WorkflowRun{Status: RunRunning, StartedAt: ptrTime(time.Now())})
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
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	rows, err := s.QueryString("SELECT id FROM crawl_tasks WHERE id = ? FOR UPDATE", taskID)
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if len(rows) == 0 {
		_ = s.Rollback()
		return nil, errors.New("task not found")
	}
	attemptNo, err := nextAttempt(taskID)
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}
	leaseUntil := time.Now().Add(5 * time.Minute)
	claimed, err := s.ID(taskID).Cols("status", "attempt_count", "lease_owner", "lease_until", "updated_at").Where("((status = ? AND (next_retry_at IS NULL OR next_retry_at <= NOW())) OR (status = ? AND lease_until < NOW())) AND attempt_count < max_attempts AND EXISTS (SELECT 1 FROM workflow_runs WHERE workflow_runs.id = crawl_tasks.run_id AND workflow_runs.status IN (?, ?))", TaskQueued, TaskRunning, RunQueued, RunRunning).Update(&table.CrawlTask{Status: TaskRunning, AttemptCount: attemptNo, LeaseOwner: workerID, LeaseUntil: &leaseUntil, UpdatedAt: time.Now()})
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if claimed == 0 {
		_ = s.Rollback()
		return nil, errors.New("task is already claimed or unavailable")
	}
	attempt := &table.TaskAttempt{TaskId: taskID, AttemptNo: attemptNo, WorkerId: workerID, Status: TaskRunning, StartedAt: ptrTime(time.Now()), RequestSnapshot: fallbackJSON(input), ResponseSnapshot: "{}"}
	if _, err := s.InsertOne(attempt); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := s.Commit(); err != nil {
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
	if attempt.TaskId != taskID || attempt.Status != TaskRunning {
		return ErrStaleAttempt
	}
	duration := now.Sub(timeOrZero(attempt.StartedAt)).Milliseconds()
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	changed, err := s.ID(taskID).Where("status = ? AND lease_owner = ? AND attempt_count = ? AND lease_until > NOW()", TaskRunning, attempt.WorkerId, attempt.AttemptNo).Cols("status", "lease_owner", "lease_until", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskSucceeded, LeaseOwner: "", LeaseUntil: nil, UpdatedAt: now, FinishedAt: &now})
	if err != nil || changed != 1 {
		_ = s.Rollback()
		if err != nil {
			return err
		}
		return ErrStaleAttempt
	}
	changed, err = s.ID(attemptID).Where("status = ?", TaskRunning).Cols("status", "finished_at", "duration_ms", "response_snapshot").Update(&table.TaskAttempt{Status: TaskSucceeded, FinishedAt: &now, DurationMs: duration, ResponseSnapshot: fallbackJSON(response)})
	if err != nil || changed != 1 {
		_ = s.Rollback()
		if err != nil {
			return err
		}
		return ErrStaleAttempt
	}
	if err := s.Commit(); err != nil {
		return err
	}
	return tryFinishRun(taskID)
}

func Fail(taskID, attemptID int64, cause error, retryable bool) (bool, error) {
	if cause == nil {
		cause = errors.New("task failed")
	}
	task := new(table.CrawlTask)
	has, err := db.Instance().ID(taskID).Get(task)
	if err != nil || !has {
		if err != nil {
			return false, err
		}
		return false, errors.New("task not found")
	}
	attempt := new(table.TaskAttempt)
	has, err = db.Instance().ID(attemptID).Get(attempt)
	if err != nil || !has {
		if err != nil {
			return false, err
		}
		return false, errors.New("task attempt not found")
	}
	if attempt.TaskId != taskID || attempt.Status != TaskRunning {
		return false, ErrStaleAttempt
	}
	now := time.Now()
	message := cause.Error()
	willRetry := retryable && task.AttemptCount < task.MaxAttempts
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return false, err
	}
	condition := s.ID(taskID).Where("status = ? AND lease_owner = ? AND attempt_count = ? AND lease_until > NOW()", TaskRunning, attempt.WorkerId, attempt.AttemptNo)
	var changed int64
	if willRetry {
		delay := backoff(task.AttemptCount)
		next := now.Add(delay)
		changed, err = condition.Cols("status", "next_retry_at", "lease_owner", "lease_until", "error_message", "updated_at").Update(&table.CrawlTask{Status: TaskQueued, NextRetryAt: &next, LeaseOwner: "", LeaseUntil: nil, ErrorMessage: message, UpdatedAt: now})
	} else {
		changed, err = condition.Cols("status", "lease_owner", "lease_until", "error_message", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskDeadLetter, LeaseOwner: "", LeaseUntil: nil, ErrorMessage: message, UpdatedAt: now, FinishedAt: &now})
	}
	if err != nil || changed != 1 {
		_ = s.Rollback()
		if err != nil {
			return false, err
		}
		return false, ErrStaleAttempt
	}
	failure := &table.TaskAttempt{Status: TaskFailed, FinishedAt: &now, ErrorMessage: message}
	columns := []string{"status", "finished_at", "error_message"}
	var evidence interface{ FailureSnapshot() any }
	if errors.As(cause, &evidence) {
		if encoded, marshalErr := json.Marshal(evidence.FailureSnapshot()); marshalErr == nil {
			failure.ResponseSnapshot = string(encoded)
			columns = append(columns, "response_snapshot")
		}
	}
	changed, err = s.ID(attemptID).Where("status = ?", TaskRunning).Cols(columns...).Update(failure)
	if err != nil || changed != 1 {
		_ = s.Rollback()
		if err != nil {
			return false, err
		}
		return false, ErrStaleAttempt
	}
	if err := s.Commit(); err != nil {
		return false, err
	}
	if willRetry {
		return true, nil
	}
	return false, tryFinishRun(taskID)
}

// RenewLease fences work by the exact attempt as well as its worker ID.
func RenewLease(taskID int64, attempt table.TaskAttempt, lease time.Duration) error {
	if lease <= 0 {
		return fmt.Errorf("lease duration must be positive")
	}
	changed, err := db.Instance().ID(taskID).Where("status = ? AND lease_owner = ? AND attempt_count = ? AND lease_until > NOW()", TaskRunning, attempt.WorkerId, attempt.AttemptNo).Cols("lease_until", "updated_at").Update(&table.CrawlTask{LeaseUntil: ptrTime(time.Now().Add(lease)), UpdatedAt: time.Now()})
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrStaleAttempt
	}
	return nil
}

func CheckAttempt(taskID int64, attempt table.TaskAttempt) error {
	has, err := db.Instance().ID(taskID).Where("status = ? AND lease_owner = ? AND attempt_count = ? AND lease_until > NOW()", TaskRunning, attempt.WorkerId, attempt.AttemptNo).Exist(new(table.CrawlTask))
	if err != nil {
		return err
	}
	if !has {
		return ErrStaleAttempt
	}
	return nil
}

func CheckAttemptID(taskID, attemptID int64) error {
	attempt := new(table.TaskAttempt)
	has, err := db.Instance().ID(attemptID).Get(attempt)
	if err != nil {
		return err
	}
	if !has || attempt.TaskId != taskID || attempt.Status != TaskRunning {
		return ErrStaleAttempt
	}
	return CheckAttempt(taskID, *attempt)
}

// WithActiveAttempt holds the task row while a result is written. A concurrent
// cancellation or reclaim must wait until the write finishes, then observes
// the new task state before it can proceed.
func WithActiveAttempt(taskID int64, attempt table.TaskAttempt, write func() error) error {
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	// Permit foreign-key KEY SHARE locks from the evidence writer's transaction
	// while still fencing cancellation/reclaim and other task state updates.
	rows, err := s.QueryString("SELECT id FROM crawl_tasks WHERE id = ? AND status = ? AND lease_owner = ? AND attempt_count = ? AND lease_until > NOW() FOR NO KEY UPDATE", taskID, TaskRunning, attempt.WorkerId, attempt.AttemptNo)
	if err != nil {
		_ = s.Rollback()
		return err
	}
	if len(rows) == 0 {
		_ = s.Rollback()
		return ErrStaleAttempt
	}
	if err := write(); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}

func WithActiveAttemptID(taskID, attemptID int64, write func() error) error {
	attempt := new(table.TaskAttempt)
	has, err := db.Instance().ID(attemptID).Get(attempt)
	if err != nil {
		return err
	}
	if !has || attempt.TaskId != taskID || attempt.Status != TaskRunning {
		return ErrStaleAttempt
	}
	return WithActiveAttempt(taskID, *attempt, write)
}

// MaintainLease renews the current attempt and cancels its context when it
// loses ownership or the task is cancelled. Call stop before finalizing it.
func MaintainLease(parent context.Context, taskID int64, attempt table.TaskAttempt, lease time.Duration) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	interval := lease / 3
	if interval > 10*time.Second {
		interval = 10 * time.Second
	}
	if interval <= 0 {
		interval = time.Second
	}
	var once sync.Once
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := RenewLease(taskID, attempt, lease); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, func() { once.Do(func() { cancel(); <-done }) }
}

func RecoverExpiredLeases() (int64, error) {
	if db.Instance() == nil {
		return 0, errors.New("database is not initialized")
	}
	now := time.Now()
	var exhausted []table.CrawlTask
	if err := db.Instance().Where("status = ? AND lease_until < NOW() AND attempt_count >= max_attempts", TaskRunning).Find(&exhausted); err != nil {
		return 0, err
	}
	for _, task := range exhausted {
		changed, err := db.Instance().ID(task.Id).Where("status = ? AND lease_until < NOW() AND attempt_count >= max_attempts", TaskRunning).Cols("status", "lease_owner", "lease_until", "error_message", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskDeadLetter, LeaseOwner: "", LeaseUntil: nil, ErrorMessage: "lease expired after maximum attempts", UpdatedAt: now, FinishedAt: &now})
		if err != nil {
			return 0, err
		}
		if changed > 0 {
			if err := tryFinishRun(task.Id); err != nil {
				return 0, err
			}
		}
	}
	result, err := db.Instance().Where("status = ? AND lease_until < NOW() AND attempt_count < max_attempts", TaskRunning).Cols("status", "next_retry_at", "lease_owner", "lease_until", "updated_at").Update(&table.CrawlTask{Status: TaskQueued, NextRetryAt: &now, LeaseOwner: "", LeaseUntil: nil, UpdatedAt: now})
	if err != nil {
		return 0, err
	}
	_, err = db.Instance().Where("status = ? AND task_id IN (SELECT id FROM crawl_tasks WHERE status IN (?, ?) AND (lease_owner IS NULL OR lease_owner = ''))", TaskRunning, TaskQueued, TaskDeadLetter).Cols("status", "finished_at", "error_message").Update(&table.TaskAttempt{Status: RunInterrupted, FinishedAt: &now, ErrorMessage: "task lease expired"})
	return result + int64(len(exhausted)), err
}

func CancelTask(id int64) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	now := time.Now()
	changed, err := s.ID(id).Where("status IN (?, ?)", TaskQueued, TaskRunning).Cols("status", "lease_owner", "lease_until", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskCancelled, LeaseOwner: "", LeaseUntil: nil, UpdatedAt: now, FinishedAt: &now})
	if err != nil || changed == 0 {
		_ = s.Rollback()
		return err
	}
	if _, err := s.Where("task_id = ? AND status = ?", id, TaskRunning).Cols("status", "finished_at").Update(&table.TaskAttempt{Status: TaskCancelled, FinishedAt: &now}); err != nil {
		_ = s.Rollback()
		return err
	}
	if err := s.Commit(); err != nil {
		return err
	}
	return tryFinishRun(id)
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
	if _, err := s.ID(id).Where("status IN (?, ?)", RunQueued, RunRunning).Cols("status", "finished_at").Update(&table.WorkflowRun{Status: RunCancelled, FinishedAt: ptrTime(time.Now())}); err != nil {
		_ = s.Rollback()
		return err
	}
	now := time.Now()
	if _, err := s.Where("run_id = ? AND status IN (?, ?)", id, TaskQueued, TaskRunning).Cols("status", "lease_owner", "lease_until", "updated_at", "finished_at").Update(&table.CrawlTask{Status: TaskCancelled, LeaseOwner: "", LeaseUntil: nil, UpdatedAt: now, FinishedAt: &now}); err != nil {
		_ = s.Rollback()
		return err
	}
	if _, err := s.Where("task_id IN (SELECT id FROM crawl_tasks WHERE run_id = ?) AND status = ?", id, TaskRunning).Cols("status", "finished_at").Update(&table.TaskAttempt{Status: TaskCancelled, FinishedAt: &now}); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}

func RetryTask(id int64) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	task := new(table.CrawlTask)
	has, err := db.Instance().ID(id).Get(task)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("task not found")
	}
	run := new(table.WorkflowRun)
	has, err = db.Instance().ID(task.RunId).Get(run)
	if err != nil {
		return err
	}
	if !has || (run.Status != RunFailed && run.Status != RunPartial && run.Status != RunRunning) {
		return errors.New("only tasks in failed, partial or running runs can be retried; rerun the workflow instead")
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	rows, err := s.QueryString("SELECT status FROM workflow_runs WHERE id = ? FOR UPDATE", task.RunId)
	if err != nil {
		_ = s.Rollback()
		return err
	}
	if len(rows) == 0 || (rows[0]["status"] != RunFailed && rows[0]["status"] != RunPartial && rows[0]["status"] != RunRunning) {
		_ = s.Rollback()
		return errors.New("workflow run is no longer retryable")
	}
	maxAttempts := task.MaxAttempts
	if task.AttemptCount >= maxAttempts {
		maxAttempts = task.AttemptCount + 5
	}
	changed, err := s.ID(id).Where("status IN (?, ?)", TaskDeadLetter, TaskCancelled).Cols("status", "max_attempts", "next_retry_at", "finished_at", "error_message", "updated_at").Update(&table.CrawlTask{Status: TaskQueued, MaxAttempts: maxAttempts, NextRetryAt: ptrTime(time.Now()), FinishedAt: nil, ErrorMessage: "", UpdatedAt: time.Now()})
	if err != nil {
		_ = s.Rollback()
		return err
	}
	if changed == 0 {
		_ = s.Rollback()
		return errors.New("only terminal tasks can be retried")
	}
	if _, err = s.ID(task.RunId).Where("status IN (?, ?, ?)", RunFailed, RunPartial, RunRunning).Cols("status", "finished_at").Update(&table.WorkflowRun{Status: RunRunning, FinishedAt: nil}); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}

func RerunRun(id int64) (*table.WorkflowRun, error) {
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	original := new(table.WorkflowRun)
	has, err := db.Instance().ID(id).Get(original)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("run not found")
	}
	now := time.Now()
	run := &table.WorkflowRun{WorkflowId: original.WorkflowId, WorkflowVersionId: original.WorkflowVersionId, TriggerType: "rerun", Status: RunQueued, Input: fallbackJSON(original.Input), Summary: "{}", CreatedBy: original.CreatedBy, CreatedAt: now}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	if _, err := s.Insert(run); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if _, err := s.Insert(&table.CrawlTask{RunId: run.Id, StepName: "trigger", TaskType: "workflow", Input: fallbackJSON(original.Input), Status: TaskQueued, MaxAttempts: 5, CreatedAt: now, UpdatedAt: now}); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := s.Commit(); err != nil {
		return nil, err
	}
	return run, nil
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
	if filter.ProjectID > 0 {
		s.Where("workflow_id IN (SELECT id FROM workflows WHERE project_id = ?)", filter.ProjectID)
	}
	if filter.WorkflowID > 0 {
		s.Where("workflow_id = ?", filter.WorkflowID)
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
	if filter.ProjectID > 0 {
		s.Where("workflow_id IN (SELECT id FROM workflows WHERE project_id = ?)", filter.ProjectID)
	}
	if filter.WorkflowID > 0 {
		s.Where("workflow_id = ?", filter.WorkflowID)
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

// UpdateRunSummary stores a compact, JSON-encoded execution result without
// changing the run state. It is intentionally separate from Complete/Fail so
// a worker can expose extracted values before the final task transition.
func UpdateRunSummary(runID int64, summary string) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	if runID <= 0 {
		return errors.New("run id is required")
	}
	return updateRunSummary(runID, fallbackJSON(summary))
}

func updateRunSummary(runID int64, summary string) error {
	_, err := db.Instance().ID(runID).Where("status IN (?, ?)", RunQueued, RunRunning).Cols("summary").Update(&table.WorkflowRun{Summary: summary})
	return err
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
		var project table.Project
		if found, err := db.Instance().Where("code = ?", "default").Get(&project); err != nil || !found {
			if err != nil {
				return 0, 0, err
			}
			return 0, 0, errors.New("default project not found")
		}
		var dataset table.Dataset
		if found, err := db.Instance().Where("project_id = ? AND code = ?", project.Id, "magnet").Get(&dataset); err != nil || !found {
			if err != nil {
				return 0, 0, err
			}
			return 0, 0, errors.New("default magnet dataset not found")
		}
		workflow = &table.Workflow{ProjectId: &project.Id, DatasetId: &dataset.Id, SourceId: sourceID, Code: "default", Name: sourceName + " default", ResourceType: "magnet", Enabled: true}
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
func tryFinishRun(taskID int64) error {
	var task table.CrawlTask
	has, err := db.Instance().ID(taskID).Get(&task)
	if err != nil || !has {
		return err
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	// Locking the run coordinates this final check with CreateTask and with
	// other workers completing tasks in the same run.
	rows, err := s.QueryString("SELECT status FROM workflow_runs WHERE id = ? FOR UPDATE", task.RunId)
	if err != nil {
		_ = s.Rollback()
		return err
	}
	if len(rows) == 0 || (rows[0]["status"] != RunQueued && rows[0]["status"] != RunRunning) {
		_ = s.Rollback()
		return nil
	}
	results, err := s.QueryString(`SELECT t.status, t.task_type, a.response_snapshot
FROM crawl_tasks t LEFT JOIN LATERAL (
 SELECT response_snapshot FROM task_attempts
 WHERE task_id = t.id AND status = 'succeeded' ORDER BY attempt_no DESC LIMIT 1
) a ON true WHERE t.run_id = ?`, task.RunId)
	if err != nil {
		_ = s.Rollback()
		return err
	}
	summary := RunSummary{}
	workflowRun := false
	for _, row := range results {
		if row["status"] == TaskQueued || row["status"] == TaskRunning {
			_ = s.Rollback()
			return nil
		}
		if row["task_type"] == "workflow" {
			workflowRun = true
		}
		summary.Add(row["status"], row["response_snapshot"])
	}
	status := aggregateRunStatus(summary.Failed > 0, summary.Cancelled > 0)
	if workflowRun {
		status = summary.Status()
	}
	update := &table.WorkflowRun{Status: status, FinishedAt: ptrTime(time.Now())}
	cols := []string{"status", "finished_at"}
	if workflowRun {
		update.Summary = summary.JSON()
		cols = append(cols, "summary")
	}
	if _, err := s.ID(task.RunId).Cols(cols...).Update(update); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}

func aggregateRunStatus(failed, cancelled bool) string {
	if failed {
		return RunFailed
	}
	if cancelled {
		return RunCancelled
	}
	return RunSucceeded
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
