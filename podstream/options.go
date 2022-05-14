package podstream

import (
	"regexp"

	"github.com/b4fun/podkit"
)

func WithLogger(logger podkit.Logger) Option {
	return func(streamer *Streamer) error {
		streamer.logger = logger
		return nil
	}
}

func FromSelectedPods(labelSelector string) Option {
	return func(streamer *Streamer) error {
		streamer.labelSelector = labelSelector
		return nil
	}
}

func FollowSelectedPods(labelSelector string) Option {
	return func(streamer *Streamer) error {
		streamer.labelSelector = labelSelector
		streamer.follow = true
		return nil
	}
}

func FromContainer(containerName string) Option {
	return func(streamer *Streamer) error {
		streamer.podLogOptions.Container = containerName

		return nil
	}
}

func ConsumeLogsWith(first LogEntryConsumer, other ...LogEntryConsumer) Option {
	consumers := append([]LogEntryConsumer{first}, other...)

	return func(streamer *Streamer) error {
		streamer.logsConsumer = LogEntryConsumers(consumers)
		return nil
	}
}

func ConsumeLogsWithFunc(first LogEntryConsumerFunc) Option {
	return func(streamer *Streamer) error {
		streamer.logsConsumer = first
		return nil
	}
}

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
