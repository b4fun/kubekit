package podstream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogEntryConsumers(t *testing.T) {
	var (
		logsConsumer LogEntryConsumers
		loadedLogs   []LogEntry
	)

	logsConsumer = append(
		logsConsumer,
		LogEntryConsumerFunc(func(logs []LogEntry) {
			loadedLogs = append(loadedLogs, logs...)
		}),
		LogEntryConsumerFunc(func(logs []LogEntry) {
			loadedLogs = append(loadedLogs, logs...)
		}),
	)

	logsConsumer.OnLogs([]LogEntry{
		{Time: time.Now(), Log: "test"},
	})

	assert.Len(t, loadedLogs, 2)
}
