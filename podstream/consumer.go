package podstream

// LogEntryConsumers combines a group of LogEntryConsumer.
// Each consumer is called one by one with with the slice order.
type LogEntryConsumers []LogEntryConsumer

var _ LogEntryConsumer = (*LogEntryConsumers)(nil)

func (s LogEntryConsumers) OnLogs(logs []LogEntry) {
	for _, c := range s {
		c.OnLogs(logs)
	}
}
