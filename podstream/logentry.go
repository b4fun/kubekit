package podstream

import "sort"

type logEntries []LogEntry

var _ sort.Interface = logEntries{}

func (le logEntries) Len() int { return len(le) }

func (le logEntries) Less(i, j int) bool {
	return le[i].Time.Before(le[j].Time)
}

func (le logEntries) Swap(i, j int) {
	le[i], le[j] = le[j], le[i]
}
