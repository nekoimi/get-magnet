package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nekoimi/scrapio/internal/bean"
	"github.com/nekoimi/scrapio/internal/config"
	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/repo/dataset_repo"
	"github.com/nekoimi/scrapio/internal/repo/record_repo"
	"github.com/nekoimi/scrapio/internal/repo/task_repo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func TestDevA04Templates(t *testing.T) {
	if os.Getenv("SCRAPIO_TEST_DEV_A04") != "1" {
		t.Skip("set SCRAPIO_TEST_DEV_A04=1 to test config/dev.yaml PostgreSQL")
	}
	previousLogLevel := log.GetLevel()
	defer log.SetLevel(previousLogLevel)
	log.SetLevel(log.ErrorLevel)
	v := viper.New()
	v.SetConfigFile("../../config/dev.yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatal("cannot read dev config")
	}
	probe, err := sql.Open("postgres", v.GetString("db.dsn"))
	if err != nil {
		t.Fatal("cannot open development database")
	}
	var databaseName string
	probeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	probeErr := probe.QueryRowContext(probeCtx, "SELECT current_database()").Scan(&databaseName)
	probe.Close()
	if probeErr != nil || databaseName != "get_magnet_dev" {
		t.Fatal("expected get_magnet_dev before initializing migrations")
	}
	ctx := bean.ContextWithDefaultRegistry(context.Background())
	bean.MustRegisterPtr(ctx, &config.Config{DB: &config.DBConfig{Dsn: v.GetString("db.dsn")}})
	lifecycle := db.NewDBLifecycle()
	if err := lifecycle.Start(ctx); err != nil {
		t.Fatal("cannot initialize database")
	}
	defer lifecycle.Stop(ctx)
	db.Instance().ShowSQL(false)
	raw := db.Instance().DB().DB
	var name string
	if err := raw.QueryRow("SELECT current_database()").Scan(&name); err != nil || name != "get_magnet_dev" {
		t.Fatal("unexpected database")
	}
	project, err := dataset_repo.CreateProject(fmt.Sprintf("a04_%d", time.Now().UnixNano()), "A04 integration", "", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		statements := []string{
			`DELETE FROM record_revisions WHERE record_id IN (SELECT r.id FROM records r JOIN datasets d ON d.id=r.dataset_id WHERE d.project_id=$1)`,
			`DELETE FROM record_observations WHERE dataset_id IN (SELECT id FROM datasets WHERE project_id=$1)`,
			`DELETE FROM records WHERE dataset_id IN (SELECT id FROM datasets WHERE project_id=$1)`,
			`DELETE FROM documents WHERE task_id IN (SELECT t.id FROM crawl_tasks t JOIN workflow_runs r ON r.id=t.run_id JOIN workflows w ON w.id=r.workflow_id WHERE w.project_id=$1)`,
			`DELETE FROM workflow_runs WHERE workflow_id IN (SELECT id FROM workflows WHERE project_id=$1)`,
			`DELETE FROM workflows WHERE project_id=$1`,
			`DELETE FROM datasets WHERE project_id=$1`,
		}
		for _, statement := range statements {
			if _, err := raw.Exec(statement, project.Id); err != nil {
				t.Error("cleanup:", err)
			}
		}
		if _, err := raw.Exec("DELETE FROM projects WHERE id=$1", project.Id); err != nil {
			t.Error(err)
		}
	}()
	dataset, err := dataset_repo.CreateDataset(dataset_repo.CreateDatasetInput{ProjectID: project.Id, Code: "article", Name: "A04 test", RecordType: "article", SchemaInput: dataset_repo.SchemaInput{UniqueKeyFields: []string{"url"}, EmptyValuePolicy: "preserve", Fields: []dataset_repo.FieldInput{{Key: "url", Label: "URL", Type: "url", Required: true}, {Key: "title", Label: "Title", Type: "string"}, {Key: "body", Label: "Body", Type: "string"}}}})
	if err != nil {
		t.Fatal(err)
	}
	magnetDataset, err := dataset_repo.CreateDataset(dataset_repo.CreateDatasetInput{ProjectID: project.Id, Code: "magnet", Name: "Magnet test", RecordType: "magnet", SchemaInput: dataset_repo.SchemaInput{UniqueKeyFields: []string{"canonical_key"}, EmptyValuePolicy: "preserve", Fields: []dataset_repo.FieldInput{{Key: "canonical_key", Label: "Key", Type: "string", Required: true}, {Key: "number", Label: "Number", Type: "string"}, {Key: "title", Label: "Title", Type: "string"}, {Key: "links", Label: "Links", Type: "string", Multiple: true}}}})
	if err != nil {
		t.Fatal(err)
	}
	var sourceID int64
	if err := raw.QueryRow("SELECT id FROM sources ORDER BY id LIMIT 1").Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/magnet" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<span class="number">AB-123</span><h1>Magnet title</h1><a href="magnet:?xt=urn:btih:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA">one</a><a href="magnet:?xt=urn:btih:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB">two</a>`)
			return
		}
		if r.URL.Path == "/page" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<link rel="canonical" href="https://a04-test.invalid/a"><h1>Old title</h1><article>Body</article>`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[{"url":"https://a04-test.invalid/a","title":"New title","body":"Body"},{"url":"https://a04-test.invalid/b","title":"Second","body":"Text"}]}`)
	}))
	defer server.Close()
	worker := NewWorker()
	execute := func(template Template, code, path string, target *table.Dataset) int64 {
		t.Helper()
		template.Definition.Trigger.URL = server.URL + path
		encoded, _ := json.Marshal(template.Definition)
		now := time.Now()
		workflow := &table.Workflow{ProjectId: &project.Id, DatasetId: &target.Id, SourceId: sourceID, Code: code, Name: code, ResourceType: target.RecordType, Enabled: true, CreatedAt: now, UpdatedAt: now}
		if _, err := db.Instance().InsertOne(workflow); err != nil {
			t.Fatal(err)
		}
		version := &table.WorkflowVersion{WorkflowId: workflow.Id, Version: 1, Status: "published", Definition: string(encoded), CreatedAt: now}
		if _, err := db.Instance().InsertOne(version); err != nil {
			t.Fatal(err)
		}
		run := &table.WorkflowRun{WorkflowId: workflow.Id, WorkflowVersionId: version.Id, TriggerType: "manual", Status: "running", Input: "{}", Summary: "{}", CreatedAt: now}
		if _, err := db.Instance().InsertOne(run); err != nil {
			t.Fatal(err)
		}
		lease := now.Add(time.Minute)
		task := table.CrawlTask{RunId: run.Id, StepName: "trigger", TaskType: "workflow", Input: "{}", Status: "running", AttemptCount: 1, MaxAttempts: 1, LeaseOwner: "a04-test", LeaseUntil: &lease, CreatedAt: now, UpdatedAt: now}
		if _, err := db.Instance().InsertOne(&task); err != nil {
			t.Fatal(err)
		}
		attempt := table.TaskAttempt{TaskId: task.Id, AttemptNo: 1, WorkerId: "a04-test", Status: "running", StartedAt: &now, RequestSnapshot: "{}", ResponseSnapshot: "{}"}
		if _, err := db.Instance().InsertOne(&attempt); err != nil {
			t.Fatal(err)
		}
		if err := worker.execute(ctx, &task_repo.Claim{Task: task, Attempt: attempt}); err != nil {
			t.Fatal("template execution:", err)
		}
		var status string
		if err := raw.QueryRow("SELECT status FROM workflow_runs WHERE id=$1", run.Id).Scan(&status); err != nil || status != "succeeded" {
			t.Fatalf("run status=%s err=%v", status, err)
		}
		return run.Id
	}
	templates := Templates()
	execute(templates[0], "page", "/page", dataset)
	jsonRun := execute(templates[1], "json", "/items", dataset)
	execute(templates[1], "json_retry", "/items", dataset)
	magnet := templates[0]
	magnet.Definition.Nodes = []Node{{Name: "extract", Type: "extract", Config: map[string]any{"content_type": "html", "fields": []any{
		map[string]any{"name": "canonical_key", "selector": ".number", "required": true},
		map[string]any{"name": "number", "selector": ".number"},
		map[string]any{"name": "title", "selector": "h1"},
		map[string]any{"name": "links", "selector": "a[href^=\"magnet:\"]", "attribute": "href", "multiple": true},
	}}}}
	magnetRun := execute(magnet, "magnet", "/magnet", magnetDataset)
	var magnetKey, magnetValues string
	if err := raw.QueryRow("SELECT canonical_key,normalized::text FROM records WHERE dataset_id=$1", magnetDataset.Id).Scan(&magnetKey, &magnetValues); err != nil {
		t.Fatal("magnet record:", err)
	}
	if magnetKey != "AB-123" || !strings.Contains(magnetValues, "magnet:?xt=urn:btih:AAAAAAAA") || !strings.Contains(magnetValues, "magnet:?xt=urn:btih:BBBBBBBB") {
		t.Fatalf("magnet values: key=%q values=%s", magnetKey, magnetValues)
	}
	var magnetObservations int
	if err := raw.QueryRow("SELECT count(*) FROM record_observations WHERE dataset_id=$1 AND run_id=$2 AND document_id IS NOT NULL", magnetDataset.Id, magnetRun).Scan(&magnetObservations); err != nil || magnetObservations != 1 {
		t.Fatalf("magnet provenance: observations=%d err=%v", magnetObservations, err)
	}
	var records, observations, revisions int
	if err := raw.QueryRow("SELECT count(*) FROM records WHERE dataset_id=$1", dataset.Id).Scan(&records); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow("SELECT count(*) FROM record_observations WHERE dataset_id=$1", dataset.Id).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow("SELECT count(*) FROM record_revisions v JOIN records r ON r.id=v.record_id WHERE r.dataset_id=$1", dataset.Id).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if records != 2 || observations != 5 || revisions != 3 {
		t.Fatalf("records=%d observations=%d revisions=%d", records, observations, revisions)
	}
	var recordID, observedRun, documentID, versionID, taskID int64
	if err := raw.QueryRow(`SELECT rec.id,ob.run_id,ob.document_id,ob.workflow_version_id,ob.task_id
		FROM records rec JOIN record_observations ob ON ob.record_id=rec.id
		WHERE rec.dataset_id=$1 AND ob.run_id=$2 ORDER BY ob.id LIMIT 1`, dataset.Id, jsonRun).Scan(&recordID, &observedRun, &documentID, &versionID, &taskID); err != nil {
		t.Fatal("record provenance:", err)
	}
	var storedType string
	if err := raw.QueryRow("SELECT document_type FROM documents WHERE id=$1 AND task_id=$2", documentID, taskID).Scan(&storedType); err != nil || storedType != "json" || observedRun != jsonRun || versionID == 0 || recordID == 0 {
		t.Fatalf("record provenance mismatch: record=%d run=%d document=%d version=%d type=%q error=%v", recordID, observedRun, documentID, versionID, storedType, err)
	}
	var summary string
	if err := raw.QueryRow("SELECT summary::text FROM workflow_runs WHERE id=$1", jsonRun).Scan(&summary); err != nil {
		t.Fatal(err)
	}
	var counts task_repo.RunSummary
	if err := json.Unmarshal([]byte(summary), &counts); err != nil || counts.Created != 1 || counts.Updated != 1 {
		t.Fatalf("summary=%s %v", summary, err)
	}
	_, err = record_repo.SaveBatch([]record_repo.Candidate{{DatasetID: dataset.Id, Values: map[string]any{"url": "https://a04-test.invalid/rollback", "title": "Good"}}, {DatasetID: dataset.Id, Values: map[string]any{"title": "Missing key"}}})
	if err == nil {
		t.Fatal("invalid batch accepted")
	}
	var rollbackCount int
	if err := raw.QueryRow("SELECT count(*) FROM records WHERE dataset_id=$1 AND canonical_key='https://a04-test.invalid/rollback'", dataset.Id).Scan(&rollbackCount); err != nil || rollbackCount != 0 {
		t.Fatal("batch did not rollback")
	}
	t.Log("HTTP HTML and JSON templates executed without browser; 2 records, 5 observations, 3 revisions; batch rollback and run summary passed")
}
