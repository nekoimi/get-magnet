package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type dailyWriter struct {
	mu      sync.Mutex
	level   logrus.Level
	baseDir string
	logger  *lumberjack.Logger
	curDate string
	writer  io.Writer
	withStd bool
}

func newDailyWriter(logDir string, level logrus.Level, withStdout bool) *dailyWriter {
	w := &dailyWriter{
		level:   level,
		baseDir: logDir,
		withStd: withStdout,
	}
	w.rotate()
	return w
}

func (w *dailyWriter) rotate() {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if w.curDate == today {
		return
	}
	w.curDate = today

	_ = os.MkdirAll(w.baseDir, 0755)
	filename := filepath.Join(w.baseDir, fmt.Sprintf("%s-%s.log", w.level.String(), today))

	w.logger = &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     10,
		Compress:   true,
	}

	if w.withStd {
		w.writer = io.MultiWriter(os.Stdout, w.logger)
	} else {
		w.writer = w.logger
	}
}

func (w *dailyWriter) Write(p []byte) (n int, err error) {
	w.rotate() // 每次写前检查是否需要切换
	return w.writer.Write(p)
}

// LevelHook 用于根据不同级别写入不同文件
type LevelHook struct {
	formatter logrus.Formatter
	writers   map[logrus.Level]io.Writer
	levels    []logrus.Level
}

// NewLevelHook 创建一个新的 hook
func NewLevelHook(logDir string, formatter logrus.Formatter) *LevelHook {
	_ = os.MkdirAll(logDir, 0755)

	hook := &LevelHook{
		formatter: formatter,
		writers:   make(map[logrus.Level]io.Writer),
		levels:    logrus.AllLevels,
	}

	// 每个级别绑定不同的 writer
	hook.writers[logrus.DebugLevel] = newDailyWriter(logDir, logrus.DebugLevel, false)
	hook.writers[logrus.InfoLevel] = newDailyWriter(logDir, logrus.InfoLevel, false)
	hook.writers[logrus.WarnLevel] = newDailyWriter(logDir, logrus.WarnLevel, false)
	hook.writers[logrus.ErrorLevel] = newDailyWriter(logDir, logrus.ErrorLevel, false)

	return hook
}

func (h *LevelHook) Levels() []logrus.Level {
	return h.levels
}

func (h *LevelHook) Fire(entry *logrus.Entry) error {
	writer, ok := h.writers[entry.Level]
	if !ok {
		return nil
	}

	msg, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	_, err = writer.Write(msg)
	return err
}
