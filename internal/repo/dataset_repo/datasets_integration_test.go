package dataset_repo

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
	"github.com/nekoimi/scrapio/internal/db/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Explicit integration test: verifies the A01 ownership and versioned-schema
// contract against the development database, without changing existing rows.
func TestDevA01ProjectDatasetSchema(t *testing.T) {
	if os.Getenv("SCRAPIO_TEST_DEV_A01") != "1" {
		t.Skip("set SCRAPIO_TEST_DEV_A01=1 to use config/dev.yaml PostgreSQL")
	}
	previous := log.GetLevel()
	defer log.SetLevel(previous)
	log.SetLevel(log.ErrorLevel)
	v := viper.New()
	v.SetConfigFile("../../../config/dev.yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	probe, err := sql.Open("postgres", v.GetString("db.dsn"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var databaseName string
	err = probe.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName)
	probe.Close()
	if err != nil || databaseName != "get_magnet_dev" {
		t.Fatal("unexpected development database")
	}
	beanCtx := bean.ContextWithDefaultRegistry(context.Background())
	bean.MustRegisterPtr(beanCtx, &config.Config{DB: &config.DBConfig{Dsn: v.GetString("db.dsn")}})
	lifecycle := db.NewDBLifecycle()
	if err := lifecycle.Start(beanCtx); err != nil {
		t.Fatal(err)
	}
	defer lifecycle.Stop(beanCtx)
	db.Instance().ShowSQL(false)
	raw := db.Instance().DB().DB
	var projectID, magnetDatasetID int64
	if err := raw.QueryRow(`SELECT p.id,d.id FROM projects p JOIN datasets d ON d.project_id=p.id WHERE p.code='default' AND d.code='magnet'`).Scan(&projectID, &magnetDatasetID); err != nil {
		t.Fatal(err)
	}
	var total, assigned, magnetMissing int64
	if err := raw.QueryRow("SELECT count(*),count(project_id) FROM workflows").Scan(&total, &assigned); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow("SELECT count(*) FROM workflows WHERE resource_type='magnet' AND (project_id<>$1 OR dataset_id<>$2)", projectID, magnetDatasetID).Scan(&magnetMissing); err != nil {
		t.Fatal(err)
	}
	if total != assigned || magnetMissing != 0 {
		t.Fatalf("workflow ownership: total=%d assigned=%d magnet_mismatches=%d", total, assigned, magnetMissing)
	}
	t.Logf("existing workflows=%d, project assigned=%d, magnet ownership mismatches=%d", total, assigned, magnetMissing)
	project, err := CreateProject(fmt.Sprintf("a01_%d", time.Now().UnixNano()), "A01 schema check", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := raw.Exec("DELETE FROM projects WHERE id=$1", project.Id); err != nil {
			t.Error(err)
		}
	}()
	schemaV1 := SchemaInput{UniqueKeyFields: []string{"url"}, EmptyValuePolicy: "preserve", Fields: []FieldInput{{Key: "url", Label: "URL", Type: "url", Required: true}, {Key: "title", Label: "Title", Type: "string"}}}
	dataset, err := CreateDataset(CreateDatasetInput{ProjectID: project.Id, Code: "article", Name: "Articles", RecordType: "article", SchemaInput: schemaV1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := raw.Exec("DELETE FROM datasets WHERE id=$1", dataset.Id); err != nil {
			t.Error(err)
		}
	}()
	schemaV2 := SchemaInput{UniqueKeyFields: []string{"url", "locale"}, EmptyValuePolicy: "overwrite", Fields: []FieldInput{{Key: "url", Label: "URL", Type: "url", Required: true}, {Key: "locale", Label: "Locale", Type: "string", Required: true}, {Key: "title", Label: "Title", Type: "string"}}}
	if _, err := UpdateSchema(dataset.Id, schemaV2); err != nil {
		t.Fatal(err)
	}
	for version, expected := range map[int]SchemaInput{1: schemaV1, 2: schemaV2} {
		row, fields, err := Detail(dataset.Id, version)
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		if err := json.Unmarshal([]byte(row.UniqueKeyFields), &keys); err != nil {
			t.Fatal(err)
		}
		if row.SchemaVersion != version || row.EmptyValuePolicy != expected.EmptyValuePolicy || len(keys) != len(expected.UniqueKeyFields) || len(fields) != len(expected.Fields) {
			t.Fatalf("schema v%d mismatch: row=%#v keys=%v fields=%#v", version, row, keys, fields)
		}
		for i, field := range fields {
			if field.FieldKey != expected.Fields[i].Key || (i < len(keys) && keys[i] != expected.UniqueKeyFields[i]) {
				t.Fatalf("schema v%d field/key %d mismatch", version, i)
			}
		}
	}
	var listed []table.Dataset
	listed, err = ListDatasets(project.Id)
	if err != nil || len(listed) != 1 || listed[0].Id != dataset.Id {
		t.Fatalf("project dataset listing: %#v %v", listed, err)
	}
}
