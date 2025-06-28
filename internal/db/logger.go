package db

import (
	log "github.com/sirupsen/logrus"
	xormlog "xorm.io/xorm/log"
)

type xormLogger struct {
}

func newXormLogger() *xormLogger {
	return &xormLogger{}
}

// Debug implement ILogger
func (s *xormLogger) Debug(v ...interface{}) {
	log.Debug(v)
}

// Debugf implement ILogger
func (s *xormLogger) Debugf(format string, v ...interface{}) {
	log.Debugf(format, v)
}

// Error implement ILogger
func (s *xormLogger) Error(v ...interface{}) {
	log.Error(v)
}

// Errorf implement ILogger
func (s *xormLogger) Errorf(format string, v ...interface{}) {
	log.Errorf(format, v)
}

// Info implement ILogger
func (s *xormLogger) Info(v ...interface{}) {
	log.Info(v)
}

// Infof implement ILogger
func (s *xormLogger) Infof(format string, v ...interface{}) {
	log.Infof(format, v)
}

// Warn implement ILogger
func (s *xormLogger) Warn(v ...interface{}) {
	log.Warn(v)
}

// Warnf implement ILogger
func (s *xormLogger) Warnf(format string, v ...interface{}) {
	log.Warnf(format, v)
}

// Level implement ILogger
func (s *xormLogger) Level() xormlog.LogLevel {
	switch log.GetLevel() {
	case log.DebugLevel:
		return xormlog.LOG_DEBUG
	case log.InfoLevel:
		return xormlog.LOG_INFO
	case log.WarnLevel:
		return xormlog.LOG_WARNING
	case log.ErrorLevel:
		return xormlog.LOG_ERR
	case log.FatalLevel:
	case log.PanicLevel:
		return xormlog.LOG_OFF
	}
	return xormlog.LOG_UNKNOWN
}

// SetLevel implement ILogger
func (s *xormLogger) SetLevel(l xormlog.LogLevel) {
}

// ShowSQL implement ILogger
func (s *xormLogger) ShowSQL(show ...bool) {
}

// IsShowSQL implement ILogger
func (s *xormLogger) IsShowSQL() bool {
	return true
}
