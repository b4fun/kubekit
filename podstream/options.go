package podstream

import (
	"regexp"

	"github.com/b4fun/podkit"
)

// WithLogger sets the logger to be used by the streamer.
func WithLogger(logger podkit.Logger) Option {
	return func(streamer *Streamer) error {
		streamer.logger = logger
		return nil
	}
}

// FromSelectedPods sets the label selector.
// It stops the streamer after all logs have been consumed.
func FromSelectedPods(labelSelector string) Option {
	return func(streamer *Streamer) error {
		streamer.labelSelector = labelSelector
		return nil
	}
}

// FollowSelectedPods sets the pod label selector and follow flag.
// It follows the log streamming until caller stops it.
func FollowSelectedPods(labelSelector string) Option {
	return func(streamer *Streamer) error {
		streamer.labelSelector = labelSelector
		streamer.follow = true
		return nil
	}
}

// FromContainer sets the container name to log from.
func FromContainer(containerName string) Option {
	return func(streamer *Streamer) error {
		streamer.podLogOptions.Container = containerName

		return nil
	}
}

// ConsumeLogsWithFunc sets the log consumer to use.
func ConsumeLogsWith(first LogEntryConsumer, other ...LogEntryConsumer) Option {
	consumers := append([]LogEntryConsumer{first}, other...)

	return func(streamer *Streamer) error {
		streamer.logsConsumer = LogEntryConsumers(consumers)
		return nil
	}
}

// ConsumeLogsWithFunc sets the log consumer to use with function.
func ConsumeLogsWithFunc(first LogEntryConsumerFunc) Option {
	return func(streamer *Streamer) error {
		streamer.logsConsumer = first
		return nil
	}
}

// FilterWithRegex filters the logs with the given regex.
func FilterWithRegex(expr string) Option {
	return func(streamer *Streamer) error {
		pattern, err := regexp.Compile(expr)
		if err != nil {
			return err
		}

		streamer.logFilter = LogFilterFunc(func(log string) bool {
			return pattern.MatchString(log)
		})

		return nil
	}
}
