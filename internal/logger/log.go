package logger

import (
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

func Initialize(logLevel string, logDir string) {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(level)
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
		PadLevelText:           true,
		CallerPrettyfier:       friendlyCaller,
	})

	// json输出：不带颜色
	jsonFormatter := &log.JSONFormatter{
		TimestampFormat:   "2006-01-02 15:04:05",
		CallerPrettyfier:  friendlyCaller,
		DisableHTMLEscape: true,
	}
	log.AddHook(newRotateHook(logDir, jsonFormatter))
}

func friendlyCaller(frame *runtime.Frame) (function string, file string) {
	return strings.ReplaceAll(frame.Function, "github.com/nekoimi/get-magnet/", " "), ""
}
