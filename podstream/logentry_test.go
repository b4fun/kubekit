package podstream

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogEntries(t *testing.T) {
	now := time.Now()
	entries := []LogEntry{
		{
			Time: now.Add(5 * time.Second),
			Log:  "now+5s",
		},
		{
			Time: now,
			Log:  "now",
		},
		{
			Time: now.Add(-5 * time.Second),
			Log:  "now-5s",
		},
	}

	sort.Sort(logEntries(entries))

	assert.Equal(t, entries[0].Log, "now-5s")
	assert.Equal(t, entries[1].Log, "now")
}
