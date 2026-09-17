package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewRotateWriterUsesRotationConfig(t *testing.T) {
	w := newRotateWriter(t.TempDir(), logrus.InfoLevel, false, RotationConfig{
		MaxSizeMB:  12,
		MaxBackups: 3,
		MaxAgeDays: 5,
		Compress:   true,
	})
	t.Cleanup(func() { _ = w.logger.Close() })

	if w.logger.MaxSize != 12 || w.logger.MaxBackups != 3 || w.logger.MaxAge != 5 || !w.logger.Compress {
		t.Fatalf("unexpected lumberjack config: %+v", w.logger)
	}
}

func TestCleanupLevelLogsAppliesAgeAndBackupLimits(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, time.September, 17, 12, 0, 0, 0, time.Local)
	files := map[string]time.Time{
		"info-2026-09-17.log":  now,
		"info-2026-09-16.log":  now.AddDate(0, 0, -1),
		"info-2026-09-15.log":  now.AddDate(0, 0, -2),
		"info-2026-09-14.log":  now.AddDate(0, 0, -3),
		"info-2026-09-01.log":  now.AddDate(0, 0, -16),
		"error-2026-09-01.log": now.AddDate(0, 0, -16),
		"info-not-managed.log": now.AddDate(0, 0, -16),
	}
	for name, modTime := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("log"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	cleanupLevelLogs(dir, logrus.InfoLevel, "info-2026-09-17.log", RotationConfig{
		MaxBackups: 2,
		MaxAgeDays: 7,
	}, now)

	for _, name := range []string{
		"info-2026-09-17.log",
		"info-2026-09-16.log",
		"info-2026-09-15.log",
		"error-2026-09-01.log",
		"info-not-managed.log",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to remain: %v", name, err)
		}
	}
	for _, name := range []string{"info-2026-09-14.log", "info-2026-09-01.log"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed, got %v", name, err)
		}
	}
}
