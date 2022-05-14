package podstream

import (
	"time"
)

// LogEntry represents a single log entry.
type LogEntry struct {
	// Time is the log time.
	Time time.Time `json:"time"`
	// Log is the log message.
	Log string `json:"log"`
}

// LogEntryConsumer consumes log entries.
type LogEntryConsumer interface {
	// OnLogs is called when new logs are available.
	OnLogs(logs []LogEntry)
}

// LogEntryConsumerFunc is a function that consumes log entries.
type LogEntryConsumerFunc func(logs []LogEntry)

func (f LogEntryConsumerFunc) OnLogs(logs []LogEntry) {
	f(logs)
}

// LogFilter filters log line.
type LogFilter interface {
	// FilterLog returns true if the log line should be consumed.
	FilterLog(log string) bool
}

// LogFilterFunc is a LogFilter that implements the FilterLog method.
type LogFilterFunc func(log string) bool

func (f LogFilterFunc) FilterLog(log string) bool {
	return f(log)
}

// Option specifies options for configuring the podstream reader.
type Option func(streamer *Streamer) error
