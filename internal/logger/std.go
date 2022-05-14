package logger

import (
	"log"
)

type StdLogger struct {
	logger *log.Logger
}

var _ Logger = (*StdLogger)(nil)

func (l *StdLogger) Log(msg string, args ...interface{}) {
	l.logger.Printf(msg, args...)
}

func NewStdLogger(l *log.Logger) *StdLogger {
	return &StdLogger{
		logger: l,
	}
}
