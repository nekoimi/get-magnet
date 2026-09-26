package workflow_repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nekoimi/scrapio/internal/bean"
	"github.com/nekoimi/scrapio/internal/config"
	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/workflow"
	"github.com/spf13/viper"
)

func TestNextScheduleTimezone(t *testing.T) {
	after := time.Date(2026, 9, 26, 0, 59, 0, 0, time.UTC)
	next, err := nextSchedule("0 9 * * *", "Asia/Shanghai", after)
	if err != nil || !next.Equal(time.Date(2026, 9, 26, 1, 0, 0, 0, time.UTC)) {
		t.Fatalf("timezone: %v %v", next, err)
	}
	for _, tc := range []struct{ spec, zone string }{{"* * *", "UTC"}, {"0 9 * * *", "bad/zone"}, {"@every 1s", "UTC"}} {
		if _, err := nextSchedule(tc.spec, tc.zone, after); err == nil {
			t.Fatalf("accepted invalid schedule: %+v", tc)
		}
	}
}

func TestDevSchedules(t *testing.T) {
	if os.Getenv("SCRAPIO_TEST_DEV_B03") != "1" {
		t.Skip("set SCRAPIO_TEST_DEV_B03=1 to test development PostgreSQL")
	}
	v := viper.New()
	v.SetConfigFile("../../../config/dev.yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	probe, err := sql.Open("postgres", v.GetString("db.dsn"))
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	var name string
	if err := probe.QueryRow("SELECT current_database()").Scan(&name); err != nil || name != "get_magnet_dev" {
		t.Fatalf("unexpected database %q: %v", name, err)
	}
	ctx := bean.ContextWithDefaultRegistry(context.Background())
	bean.MustRegisterPtr(ctx, &config.Config{DB: &config.DBConfig{Dsn: v.GetString("db.dsn")}})
	life := db.NewDBLifecycle()
	if err := life.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer life.Stop(ctx)
	db.Instance().ShowSQL(false)
	// Remove a stale fixture left by a previously interrupted B03 test run.
	if _, err := probe.Exec("DELETE FROM workflow_runs WHERE workflow_id IN (SELECT id FROM workflows WHERE code LIKE 'b03_%' AND name='B03 schedule test')"); err != nil {
		t.Fatal(err)
	}
	if _, err := probe.Exec("DELETE FROM workflows WHERE code LIKE 'b03_%' AND name='B03 schedule test'"); err != nil {
		t.Fatal(err)
	}
	var projectID, datasetID, sourceID int64
	if err := probe.QueryRow("SELECT project_id,id FROM datasets WHERE code='article' ORDER BY id LIMIT 1").Scan(&projectID, &datasetID); err != nil {
		t.Fatal(err)
	}
	if err := probe.QueryRow("SELECT id FROM sources ORDER BY id LIMIT 1").Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	definition, _ := json.Marshal(workflow.Templates()[0].Definition)
	owner, version, err := Create(CreateWorkflowInput{ProjectID: &projectID, DatasetID: &datasetID, SourceID: &sourceID, Code: fmt.Sprintf("b03_%d", time.Now().UnixNano()), Name: "B03 schedule test", ResourceType: "article", Definition: string(definition)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := probe.Exec("DELETE FROM workflow_runs WHERE workflow_id=$1", owner.Id); err != nil {
			t.Error(err)
		}
		if _, err := probe.Exec("DELETE FROM workflows WHERE id=$1", owner.Id); err != nil {
			t.Error(err)
		}
	}()
	_, err = SaveSample(SampleInput{VersionID: version.Id, Source: "paste", PageURL: "https://example.org/b03", ContentType: "html", Content: `<link rel="canonical" href="https://example.org/b03"><h1>Title</h1><article>Body</article>`})
	if err != nil {
		t.Fatal(err)
	}
	if err := PublishVersion(version.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveSchedule(Schedule{WorkflowID: owner.Id, Cron: "* * * * *", Timezone: "UTC", Enabled: true, ConcurrencyPolicy: "skip"}); err != nil {
		t.Fatal(err)
	}
	manual, _, err := StartRun(owner.Id, "{}", nil)
	if err != nil || manual.TriggerType != "manual" {
		t.Fatalf("manual: %v %v", manual, err)
	}
	api, _, err := StartRunWithTrigger(owner.Id, "{}", nil, "api")
	if err != nil || api.TriggerType != "api" {
		t.Fatalf("api: %v %v", api, err)
	}
	if _, _, err := StartRunWithTrigger(owner.Id, "{}", nil, "webhook"); err == nil {
		t.Fatal("webhook accepted")
	}
	if _, err := probe.Exec("UPDATE workflow_schedules SET next_run_at=NOW()-INTERVAL '2 minutes' WHERE workflow_id=$1", owner.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := DispatchScheduleTick(time.Now()); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := probe.QueryRow("SELECT status FROM workflow_schedule_events WHERE workflow_id=$1 ORDER BY id DESC LIMIT 1", owner.Id).Scan(&status); err != nil || status != "skipped" {
		t.Fatalf("skip: %s %v", status, err)
	}
	if _, err := SaveSchedule(Schedule{WorkflowID: owner.Id, Cron: "* * * * *", Timezone: "UTC", Enabled: true, ConcurrencyPolicy: "queue"}); err != nil {
		t.Fatal(err)
	}
	if _, err := probe.Exec("UPDATE workflow_schedules SET next_run_at=NOW()-INTERVAL '1 minute' WHERE workflow_id=$1", owner.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := DispatchScheduleTick(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := probe.QueryRow("SELECT status FROM workflow_schedule_events WHERE workflow_id=$1 ORDER BY id DESC LIMIT 1", owner.Id).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("queue: %s %v", status, err)
	}
	// Repeated polling (and a fresh scheduler process) sees the same pending event.
	if _, err := DispatchScheduleTick(time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := probe.Exec("UPDATE workflow_runs SET status='succeeded',finished_at=NOW() WHERE workflow_id=$1", owner.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := DispatchScheduleTick(time.Now()); err != nil {
		t.Fatal(err)
	}
	var trigger string
	if err := probe.QueryRow("SELECT e.status,r.trigger_type FROM workflow_schedule_events e JOIN workflow_runs r ON r.id=e.run_id WHERE e.workflow_id=$1 ORDER BY e.id DESC LIMIT 1", owner.Id).Scan(&status, &trigger); err != nil || status != "started" || trigger != "cron" {
		t.Fatalf("dispatched: %s %s %v", status, trigger, err)
	}
	if _, err := DispatchScheduleTick(time.Now()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := probe.QueryRow("SELECT count(*) FROM workflow_runs WHERE workflow_id=$1 AND trigger_type='cron'", owner.Id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate runs: %d %v", count, err)
	}
	if err := Stop(owner.Id); err != nil {
		t.Fatal(err)
	}
	row, _, err := GetSchedule(owner.Id)
	if err != nil || row.Enabled {
		t.Fatalf("stop did not disable schedule: %+v %v", row, err)
	}
}
