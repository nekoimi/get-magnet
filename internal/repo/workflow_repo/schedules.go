package workflow_repo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nekoimi/scrapio/internal/db"
	"github.com/robfig/cron/v3"
	"xorm.io/xorm"
)

type Schedule struct {
	WorkflowID        int64      `xorm:"workflow_id pk" json:"workflow_id"`
	Cron              string     `json:"cron"`
	Timezone          string     `json:"timezone"`
	Enabled           bool       `json:"enabled"`
	ConcurrencyPolicy string     `xorm:"concurrency_policy" json:"concurrency_policy"`
	NextRunAt         *time.Time `xorm:"next_run_at" json:"next_run_at,omitempty"`
	LastRunAt         *time.Time `xorm:"last_run_at" json:"last_run_at,omitempty"`
	CreatedAt         time.Time  `xorm:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `xorm:"updated_at" json:"updated_at"`
}

func (Schedule) TableName() string { return "workflow_schedules" }

type ScheduleEvent struct {
	ID          int64     `xorm:"id" json:"id"`
	WorkflowID  int64     `xorm:"workflow_id" json:"workflow_id"`
	ScheduledAt time.Time `xorm:"scheduled_at" json:"scheduled_at"`
	Status      string    `json:"status"`
	RunID       *int64    `xorm:"run_id" json:"run_id,omitempty"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `xorm:"created_at" json:"created_at"`
}

func (ScheduleEvent) TableName() string { return "workflow_schedule_events" }

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func nextSchedule(spec, zone string, after time.Time) (time.Time, error) {
	if strings.TrimSpace(spec) != spec || spec == "" {
		return time.Time{}, errors.New("invalid cron expression")
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone: %w", err)
	}
	expression, err := cronParser.Parse(spec)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}
	next := expression.Next(after.In(loc))
	if next.IsZero() {
		return time.Time{}, errors.New("cron has no next run")
	}
	return next.UTC(), nil
}

func GetSchedule(workflowID int64) (*Schedule, bool, error) {
	row := new(Schedule)
	has, err := db.Instance().ID(workflowID).Get(row)
	return row, has, err
}

func ListScheduleEvents(workflowID int64) ([]ScheduleEvent, error) {
	rows := make([]ScheduleEvent, 0)
	err := db.Instance().Where("workflow_id=?", workflowID).Desc("id").Limit(20).Find(&rows)
	return rows, err
}

func SaveSchedule(input Schedule) (*Schedule, error) {
	if input.WorkflowID <= 0 || (input.ConcurrencyPolicy != "skip" && input.ConcurrencyPolicy != "queue") {
		return nil, errors.New("invalid schedule configuration")
	}
	if input.Timezone == "" {
		input.Timezone = "UTC"
	}
	next, err := nextSchedule(input.Cron, input.Timezone, time.Now())
	if err != nil {
		return nil, err
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	if _, err := s.QueryString("SELECT workflow_id FROM workflow_schedules WHERE workflow_id=? FOR UPDATE", input.WorkflowID); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if _, err := s.QueryString("SELECT id FROM workflows WHERE id=? FOR UPDATE", input.WorkflowID); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	row, has, err := Get(input.WorkflowID)
	if err != nil || !has {
		_ = s.Rollback()
		return nil, errors.New("workflow not found")
	}
	if input.Enabled && (!row.Enabled || row.PublishedVersionId == nil) {
		_ = s.Rollback()
		return nil, errors.New("publish and enable the workflow before scheduling")
	}
	if input.Enabled {
		input.NextRunAt = &next
	} else {
		input.NextRunAt = nil
	}
	input.UpdatedAt = time.Now()
	_, err = s.Exec(`INSERT INTO workflow_schedules (workflow_id,cron,timezone,enabled,concurrency_policy,next_run_at,updated_at)
VALUES (?,?,?,?,?,?,NOW()) ON CONFLICT (workflow_id) DO UPDATE SET cron=EXCLUDED.cron,timezone=EXCLUDED.timezone,
enabled=EXCLUDED.enabled,concurrency_policy=EXCLUDED.concurrency_policy,next_run_at=EXCLUDED.next_run_at,updated_at=NOW()`, input.WorkflowID, input.Cron, input.Timezone, input.Enabled, input.ConcurrencyPolicy, input.NextRunAt)
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}
	// Editing or disabling a schedule discards old queued ticks; their evidence remains.
	if _, err = s.Exec("UPDATE workflow_schedule_events SET status='skipped',reason='schedule changed' WHERE workflow_id=? AND status='pending'", input.WorkflowID); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := s.Commit(); err != nil {
		return nil, err
	}
	return GetScheduleRow(input.WorkflowID)
}

func GetScheduleRow(id int64) (*Schedule, error) {
	row, has, err := GetSchedule(id)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("schedule not found")
	}
	return row, nil
}

// DispatchScheduleTick commits a due tick and at most one queued run atomically.
// The database row lock lets multiple processes safely poll the same schedule.
func DispatchScheduleTick(now time.Time) (bool, error) {
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return false, err
	}
	rows, err := s.QueryString(`SELECT workflow_id FROM workflow_schedules WHERE enabled AND
 (next_run_at<=? OR (EXISTS
 (SELECT 1 FROM workflow_schedule_events e WHERE e.workflow_id=workflow_schedules.workflow_id AND e.status='pending')
 AND (concurrency_policy='skip' OR NOT EXISTS
 (SELECT 1 FROM workflow_runs r WHERE r.workflow_id=workflow_schedules.workflow_id AND r.status IN ('queued','running')))))
 ORDER BY next_run_at NULLS LAST,workflow_id LIMIT 1 FOR UPDATE SKIP LOCKED`, now)
	if err != nil {
		_ = s.Rollback()
		return false, err
	}
	if len(rows) == 0 {
		_ = s.Rollback()
		return false, nil
	}
	var schedule Schedule
	if _, err = s.ID(rows[0]["workflow_id"]).Get(&schedule); err != nil {
		_ = s.Rollback()
		return false, err
	}
	if _, err = s.QueryString("SELECT id FROM workflows WHERE id=? FOR UPDATE", schedule.WorkflowID); err != nil {
		_ = s.Rollback()
		return false, err
	}
	dueTick := schedule.NextRunAt != nil && !schedule.NextRunAt.After(now)
	if dueTick {
		due := *schedule.NextRunAt
		// One catch-up tick after downtime; do not replay an unbounded backlog.
		anchor := due
		if now.After(anchor) {
			anchor = now
		}
		next, nextErr := nextSchedule(schedule.Cron, schedule.Timezone, anchor)
		if nextErr != nil {
			_ = s.Rollback()
			return false, nextErr
		}
		if _, err = s.Exec("UPDATE workflow_schedules SET next_run_at=?,last_run_at=?,updated_at=NOW() WHERE workflow_id=?", next, due, schedule.WorkflowID); err != nil {
			_ = s.Rollback()
			return false, err
		}
		if _, err = s.Exec("INSERT INTO workflow_schedule_events(workflow_id,scheduled_at,status) VALUES (?,?,'pending') ON CONFLICT DO NOTHING", schedule.WorkflowID, due); err != nil {
			_ = s.Rollback()
			return false, err
		}
	}
	active, err := activeRuns(s, schedule.WorkflowID)
	if err != nil {
		_ = s.Rollback()
		return false, err
	}
	deferred := active > 0 && schedule.ConcurrencyPolicy == "queue"
	var pending ScheduleEvent
	has, err := s.Where("workflow_id=? AND status='pending'", schedule.WorkflowID).Asc("id").Get(&pending)
	if err != nil {
		_ = s.Rollback()
		return false, err
	}
	if has && active > 0 && schedule.ConcurrencyPolicy == "skip" {
		_, err = s.Exec("UPDATE workflow_schedule_events SET status='skipped',reason='active run' WHERE id=?", pending.ID)
	} else if has && active == 0 {
		var runID *int64
		run, _, runErr := startRunTx(s, schedule.WorkflowID, "{}", nil, "cron")
		status, reason := "started", ""
		if runErr != nil {
			status, reason = "failed", runErr.Error()
		} else {
			runID = &run.Id
		}
		_, err = s.Exec("UPDATE workflow_schedule_events SET status=?,run_id=?,reason=? WHERE id=?", status, runID, reason, pending.ID)
	}
	if err != nil {
		_ = s.Rollback()
		return false, err
	}
	if err := s.Commit(); err != nil {
		return false, err
	}
	return dueTick || !deferred, nil
}

func activeRuns(s *xorm.Session, workflowID int64) (int64, error) {
	rows, err := s.QueryString("SELECT COUNT(*) AS n FROM workflow_runs WHERE workflow_id=? AND status IN ('queued','running')", workflowID)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	var count int64
	_, err = fmt.Sscan(rows[0]["n"], &count)
	return count, err
}
