package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const defaultMaxSizeMB = 20

type RotationConfig struct {
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

func (c RotationConfig) normalized() RotationConfig {
	if c.MaxSizeMB <= 0 {
		c.MaxSizeMB = defaultMaxSizeMB
	}
	if c.MaxBackups < 0 {
		c.MaxBackups = 0
	}
	if c.MaxAgeDays < 0 {
		c.MaxAgeDays = 0
	}
	return c
}

type rotateWriter struct {
	mu      sync.Mutex
	level   logrus.Level
	baseDir string
	withStd bool
	config  RotationConfig
	logger  *lumberjack.Logger
	curDate string
	writer  io.Writer
}

func newRotateWriter(logDir string, level logrus.Level, withStdout bool, config RotationConfig) *rotateWriter {
	w := &rotateWriter{
		level:   level,
		baseDir: logDir,
		withStd: withStdout,
		config:  config.normalized(),
		writer:  io.Discard,
	}
	w.rotateLocked(time.Now())
	return w
}

func (w *rotateWriter) rotateLocked(now time.Time) {
	today := now.Format("2006-01-02")
	if w.curDate == today {
		return
	}
	if w.logger != nil {
		_ = w.logger.Close()
	}
	w.curDate = today

	_ = os.MkdirAll(w.baseDir, 0755)
	filename := filepath.Join(w.baseDir, fmt.Sprintf("%s-%s.log", w.level.String(), today))

	w.logger = &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    w.config.MaxSizeMB,
		MaxBackups: w.config.MaxBackups,
		MaxAge:     w.config.MaxAgeDays,
		Compress:   w.config.Compress,
	}

	if w.withStd {
		w.writer = io.MultiWriter(os.Stdout, w.logger)
	} else {
		w.writer = w.logger
	}

	cleanupLevelLogs(w.baseDir, w.level, filepath.Base(filename), w.config, now)
}

func (w *rotateWriter) support(level logrus.Level) bool {
	return w.level >= level
}

func (w *rotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rotateLocked(time.Now())
	return w.writer.Write(p)
}

type logFile struct {
	name    string
	modTime time.Time
}

func cleanupLevelLogs(logDir string, level logrus.Level, activeFile string, config RotationConfig, now time.Time) {
	if config.MaxBackups == 0 && config.MaxAgeDays == 0 {
		return
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(level.String()) + `-\d{4}-\d{2}-\d{2}(?:-.*)?\.log(?:\.gz)?$`)
	files := make([]logFile, 0)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == activeFile || !pattern.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, logFile{name: entry.Name(), modTime: info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].modTime.Equal(files[j].modTime) {
			return files[i].name > files[j].name
		}
		return files[i].modTime.After(files[j].modTime)
	})

	var oldestAllowed time.Time
	if config.MaxAgeDays > 0 {
		oldestAllowed = now.AddDate(0, 0, -config.MaxAgeDays)
	}
	kept := 0
	for _, file := range files {
		expired := !oldestAllowed.IsZero() && file.modTime.Before(oldestAllowed)
		overLimit := config.MaxBackups > 0 && kept >= config.MaxBackups
		if expired || overLimit {
			_ = os.Remove(filepath.Join(logDir, file.name))
			continue
		}
		kept++
	}
}

// RotateHook 用于根据不同级别写入不同文件
type RotateHook struct {
	formatter logrus.Formatter
	writers   []*rotateWriter
	levels    []logrus.Level
}

// 创建一个新的 hook
func newRotateHook(logDir string, formatter logrus.Formatter, config RotationConfig) *RotateHook {
	_ = os.MkdirAll(logDir, 0755)

	hook := &RotateHook{
		formatter: formatter,
		writers:   make([]*rotateWriter, 0),
		levels:    logrus.AllLevels,
	}

	// 每个级别绑定不同的 writer
	for _, level := range hook.levels {
		hook.writers = append(hook.writers, newRotateWriter(logDir, level, false, config))
	}

	return hook
}

func (h *RotateHook) Levels() []logrus.Level {
	return h.levels
}

func (h *RotateHook) Fire(entry *logrus.Entry) error {
	msg, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	for _, writer := range h.writers {
		if writer.support(entry.Level) {
			_, err = writer.Write(msg)
		}
	}

	return err
}
