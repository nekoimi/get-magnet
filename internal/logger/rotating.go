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

type rotateWriter struct {
	mu      *sync.Mutex
	level   logrus.Level
	baseDir string
	withStd bool
	logger  *lumberjack.Logger
	curDate string
	writer  io.Writer
}

func newRotateWriter(logDir string, level logrus.Level, withStdout bool) *rotateWriter {
	w := &rotateWriter{
		mu:      &sync.Mutex{},
		level:   level,
		baseDir: logDir,
		withStd: withStdout,
	}
	w.rotate()
	return w
}

func (w *rotateWriter) rotate() {
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

func (w *rotateWriter) Support(level logrus.Level) bool {
	return w.level >= level
}

func (w *rotateWriter) Write(p []byte) (n int, err error) {
	w.rotate() // 每次写前检查是否需要切换
	return w.writer.Write(p)
}

// RotateHook 用于根据不同级别写入不同文件
type RotateHook struct {
	formatter logrus.Formatter
	writers   []*rotateWriter
	levels    []logrus.Level
}

// 创建一个新的 hook
func newRotateHook(logDir string, formatter logrus.Formatter) *RotateHook {
	_ = os.MkdirAll(logDir, 0755)

	hook := &RotateHook{
		formatter: formatter,
		writers:   make([]*rotateWriter, 0),
		levels:    logrus.AllLevels,
	}

	// 每个级别绑定不同的 writer
	for _, level := range hook.levels {
		hook.writers = append(hook.writers, newRotateWriter(logDir, level, false))
	}

	return hook
}

func (h *RotateHook) Levels() []logrus.Level {
	return h.levels
}

func (h *RotateHook) Fire(entry *logrus.Entry) error {
	for _, writer := range h.writers {
		if writer.Support(entry.Level) {
			msg, err := h.formatter.Format(entry)
			if err != nil {
				return err
			}

			_, err = writer.Write(msg)
			return err
		}
	}

	return nil
}
