package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactedMasksSecretsWithoutMutatingSource(t *testing.T) {
	source := &Config{
		JwtSecret: "jwt-secret-value",
		Aria2:     &Aria2Config{Secret: "aria-secret"},
		DB:        &DBConfig{Dsn: "postgres://user:password@db:5432/app"},
		QuickAPI:  &QuickAPIConfig{Token: "quick-token"},
	}
	safe := source.Redacted()
	serialized := safe.String()
	for _, secret := range []string{"jwt-secret-value", "aria-secret", "password", "quick-token"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("redacted config contains secret %q: %s", secret, serialized)
		}
	}
	if source.JwtSecret != "jwt-secret-value" || source.Aria2.Secret != "aria-secret" || source.QuickAPI.Token != "quick-token" {
		t.Fatal("Redacted mutated source config")
	}
}

func TestLoadDefaultsAndEnvironmentOverrides(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "test.yaml")
	if err := os.WriteFile(configPath, []byte("log_level: error\nlog_dir: \""+filepath.ToSlash(t.TempDir())+"\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", configPath)
	t.Setenv("CRAWLER_WORKER_NUM", "7")
	t.Setenv("LOG_ROTATION_MAX_SIZE_MB", "12")
	cfg := Load()
	if cfg.Port != 8093 {
		t.Fatalf("default port = %d; want 8093", cfg.Port)
	}
	if cfg.Download != nil || cfg.CloudDriver != nil || cfg.STRM != nil || cfg.Aria2 != nil {
		t.Fatalf("delivery config should not be loaded by the v2 control plane: %+v", cfg)
	}
	if cfg.Crawler == nil || cfg.Crawler.WorkerNum != 7 {
		t.Fatalf("unexpected crawler config: %+v", cfg.Crawler)
	}
	if cfg.LogRotation.MaxSizeMB != 12 || cfg.LogRotation.MaxBackups != 7 || cfg.LogRotation.MaxAgeDays != 7 || !cfg.LogRotation.Compress {
		t.Fatalf("unexpected log rotation config: %+v", cfg.LogRotation)
	}
}
