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
		if err := PublishVersion(version.Id); err != nil {
			t.Fatal("non-magnet publication:", err)
		}
		run, task, err := StartRun(owner.Id, "{}", nil)
		if err != nil || run == nil || task == nil {
			t.Fatalf("start published template: %v", err)
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
