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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func TestDevRecordTemplatePublication(t *testing.T) {
	if os.Getenv("SCRAPIO_TEST_DEV_A04") != "1" {
		t.Skip("set SCRAPIO_TEST_DEV_A04=1 to test config/dev.yaml PostgreSQL")
	}
	previous := log.GetLevel()
	defer log.SetLevel(previous)
	log.SetLevel(log.ErrorLevel)
	v := viper.New()
	v.SetConfigFile("../../../config/dev.yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatal("cannot read dev config")
	}
	probe, err := sql.Open("postgres", v.GetString("db.dsn"))
	if err != nil {
		t.Fatal("cannot open development database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var name string
	err = probe.QueryRowContext(ctx, "SELECT current_database()").Scan(&name)
	probe.Close()
	if err != nil || name != "get_magnet_dev" {
		t.Fatal("unexpected development database")
	}
	beanCtx := bean.ContextWithDefaultRegistry(context.Background())
	bean.MustRegisterPtr(beanCtx, &config.Config{DB: &config.DBConfig{Dsn: v.GetString("db.dsn")}})
	lifecycle := db.NewDBLifecycle()
	if err := lifecycle.Start(beanCtx); err != nil {
		t.Fatal("cannot initialize database")
	}
	defer lifecycle.Stop(beanCtx)
	db.Instance().ShowSQL(false)
	raw := db.Instance().DB().DB
	var projectID, datasetID, sourceID int64
	if err := raw.QueryRow("SELECT d.project_id,d.id FROM datasets d JOIN projects p ON p.id=d.project_id WHERE p.code='default' AND d.code='article'").Scan(&projectID, &datasetID); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow("SELECT id FROM sources ORDER BY id LIMIT 1").Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	for _, template := range workflow.Templates() {
		encoded, _ := json.Marshal(template.Definition)
		owner, version, err := Create(CreateWorkflowInput{ProjectID: &projectID, DatasetID: &datasetID, SourceID: &sourceID, Code: fmt.Sprintf("a04_publish_%d", time.Now().UnixNano()), Name: "A04 publication test", ResourceType: "article", Definition: string(encoded)})
		if err != nil {
			t.Fatal(err)
		}
		defer func(id int64) {
			if _, err := raw.Exec("DELETE FROM workflow_runs WHERE workflow_id=$1", id); err != nil {
				t.Error(err)
			}
			if _, err := raw.Exec("DELETE FROM workflows WHERE id=$1", id); err != nil {
				t.Error(err)
			}
		}(owner.Id)
		if err := PublishVersion(version.Id); err == nil {
			t.Fatal("records draft published without a sample")
		}
		url := fmt.Sprintf("https://example.org/a05/%d", time.Now().UnixNano())
		contentType, content := "html", fmt.Sprintf(`<link rel="canonical" href="%s"><h1> First </h1><article>Body</article>`, url)
		if template.Code == "json_api" {
			contentType = "json"
			content = fmt.Sprintf(`{"items":[{"url":%q,"title":"First"},{"url":%q,"title":"Second"}]}`, url, url)
		}
		sample, err := SaveSample(SampleInput{VersionID: version.Id, Source: "paste", PageURL: url, ContentType: contentType, Content: content})
		if err != nil {
			t.Fatal(err)
		}
		var before, after int
		var observationsBefore, observationsAfter, revisionsBefore, revisionsAfter, pluginBefore, pluginAfter int
		if err := raw.QueryRow("SELECT count(*) FROM records WHERE dataset_id=$1", datasetID).Scan(&before); err != nil {
			t.Fatal(err)
		}
		_ = raw.QueryRow("SELECT count(*) FROM record_observations WHERE dataset_id=$1", datasetID).Scan(&observationsBefore)
		_ = raw.QueryRow("SELECT count(*) FROM record_revisions").Scan(&revisionsBefore)
		_ = raw.QueryRow("SELECT count(*) FROM plugin_tasks").Scan(&pluginBefore)
		preview, err := PreviewSample(version.Id, sample)
		if err != nil || !preview.Passed {
			t.Fatalf("sample dry-run: %#v %v", preview, err)
		}
		if template.Code == "json_api" && (len(preview.Decisions) != 2 || preview.Decisions[0].Decision != "created" || preview.Decisions[1].Decision != "updated") {
			t.Fatalf("batch decisions: %#v", preview.Decisions)
		}
		if err := raw.QueryRow("SELECT count(*) FROM records WHERE dataset_id=$1", datasetID).Scan(&after); err != nil || before != after {
			t.Fatalf("dry-run wrote records: %d %d %v", before, after, err)
		}
		_ = raw.QueryRow("SELECT count(*) FROM record_observations WHERE dataset_id=$1", datasetID).Scan(&observationsAfter)
		_ = raw.QueryRow("SELECT count(*) FROM record_revisions").Scan(&revisionsAfter)
		_ = raw.QueryRow("SELECT count(*) FROM plugin_tasks").Scan(&pluginAfter)
		if observationsBefore != observationsAfter || revisionsBefore != revisionsAfter || pluginBefore != pluginAfter {
			t.Fatal("dry-run wrote observations, revisions or plugin tasks")
		}
		if _, err := CheckSamples(version.Id); err != nil {
			t.Fatal(err)
		}
		if err := DeleteSample(sample.Id); err != nil {
			t.Fatal(err)
		}
		if err := PublishVersion(version.Id); err == nil {
			t.Fatal("deleted sample did not block publication")
		}
		sample, err = SaveSample(SampleInput{VersionID: version.Id, Source: "paste", PageURL: url, ContentType: contentType, Content: content})
		if err != nil {
			t.Fatal(err)
		}
		if err := PublishVersion(version.Id); err != nil {
			t.Fatal("non-magnet publication:", err)
		}
		if err := DeleteSample(sample.Id); err == nil {
			t.Fatal("published sample was deleted")
		}
		run, task, err := StartRun(owner.Id, "{}", nil)
		if err != nil || run == nil || task == nil {
			t.Fatalf("start published template: %v", err)
		}
		validDraft, err := CreateDraft(owner.Id, string(encoded), nil)
		if err != nil {
			t.Fatal(err)
		}
		var documentID int64
		if err := raw.QueryRow(`INSERT INTO documents(task_id,document_type,content,content_size,metadata) VALUES($1,$2,$3,$4,'{}'::jsonb) RETURNING id`, task.Id, contentType, content, len(content)).Scan(&documentID); err != nil {
			t.Fatal(err)
		}
		defer func() { _, _ = raw.Exec("DELETE FROM documents WHERE id=$1", documentID) }()
		if _, err := SaveSample(SampleInput{VersionID: validDraft.Id, Source: "document", DocumentID: &documentID, ContentType: contentType, Content: "tampered"}); err == nil {
			t.Fatal("tampered historical document accepted")
		}
		history, err := SaveSample(SampleInput{VersionID: validDraft.Id, Source: "document", DocumentID: &documentID, ContentType: contentType, Content: content})
		if err != nil {
			t.Fatal(err)
		}
		if result, err := PreviewSample(validDraft.Id, history); err != nil || !result.Passed {
			t.Fatalf("historical preview: %#v %v", result, err)
		}
		definition := template.Definition
		definition.Nodes = append(definition.Nodes, workflow.Node{Name: "remove_key", Type: "transform", Config: map[string]any{"operations": []any{map[string]any{"op": "delete", "field": "url"}}}})
		invalid, _ := json.Marshal(definition)
		draft, err := CreateDraft(owner.Id, string(invalid), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := PublishVersion(draft.Id); err == nil {
			t.Fatal("published template without dataset key")
		}
	}
	t.Log("both non-magnet templates published and created durable runs; missing unique key blocked publication")
}
