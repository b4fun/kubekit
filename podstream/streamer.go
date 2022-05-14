package podstream

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/b4fun/podkit/internal/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Stream starts the pod stream.
// It stops when the stop channel returned or any terminal error occurs.
func Stream(
	stop <-chan struct{},
	podsClient typedcorev1.PodInterface,
	options ...Option,
) error {
	streamer := &Streamer{
		logsConsumer: LogEntryConsumers{},
	}
	for _, opt := range options {
		if err := opt(streamer); err != nil {
			return err
		}
	}
	streamer.podsClient = podsClient
	if streamer.logger == nil {
		streamer.logger = logger.NoOp
	}
	if streamer.emitLogsInterval < 1 {
		streamer.emitLogsInterval = 1 * time.Second
	}

	return streamer.start(stop)
}

type Streamer struct {
	logger logger.Logger

	// podsClient is the client used to fetch pods.
	podsClient typedcorev1.PodInterface

	// podLogOptions specifies the options for fetching pod logs.
	podLogOptions corev1.PodLogOptions

	// follow indicates whether the reader should follow the pod logs.
	follow bool

	// labelSelector specifies the pods label selector to use.
	labelSelector string

	// logFilter specifies the log filter to use.
	logFilter LogFilter

	// logsConsumer specifies the logs consumer to use.
	logsConsumer LogEntryConsumer

	// emitLogsInterface speicifies the interval for emitting logs.
	emitLogsInterval time.Duration
}

func (s *Streamer) podsListOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: s.labelSelector,
	}
}

func (s *Streamer) start(stop <-chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-stop
		cancel()
	}()

	buf := make(chan LogEntry, 128)

	knownPods := map[types.UID]struct{}{}
	knownPodsLock := &sync.Mutex{}

	var podWorks sync.WaitGroup

	// trackPod attempts to put the pod into log stream tracking
	trackPod := func(pod *corev1.Pod) {
		if pod.Status.Phase == corev1.PodPending {
			// the pod is pending to be scheduled, skip it
			return
		}

		knownPodsLock.Lock()
		defer knownPodsLock.Unlock()

		if _, exists := knownPods[pod.UID]; exists {
			// the pod is already tracked, skip it
			return
		}

		podWorks.Add(1)
		go func(podName string) {
			defer podWorks.Done()
			s.streamPod(ctx.Done(), podName, buf)
		}(pod.GetName())
		knownPods[pod.UID] = struct{}{}
	}

	s.logger.Log("listing pods")
	podsList, err := s.podsClient.List(ctx, s.podsListOptions())
	if err != nil {
		err = fmt.Errorf("list pods: %w", err)
		s.logger.Log(err.Error())
		return err
	}
	sort.Slice(podsList.Items, func(i, j int) bool {
		return podsList.Items[i].Status.StartTime.Before(podsList.Items[j].Status.StartTime)
	})
	for idx := range podsList.Items {
		trackPod(&podsList.Items[idx])
	}

	if s.follow {
		go s.watch(ctx, trackPod)
	}

	consumeWork := make(chan struct{})
	go func() {
		defer close(consumeWork)

		s.consumeLogs(ctx, buf)
	}()

	if s.follow {
		// in follow mode, wait until caller cancel
		<-ctx.Done()
		s.logger.Log("caller has cancelled the stream")
	} else {
		// in non-follow mode, stop once all pods are tracked
		podWorks.Wait()
		s.logger.Log("pod workers have stopped")
		cancel()
	}
	<-consumeWork
	s.logger.Log("consume worker has stopped")

	return nil
}

func (s *Streamer) watch(ctx context.Context, trackPods func(pod *corev1.Pod)) {
	s.logger.Log("watching pods")
	defer s.logger.Log("watch worker has stopped")

	podsListOptions := s.podsListOptions()

	watchPods := func() (watch.Interface, error) {
		return s.podsClient.Watch(ctx, podsListOptions)
	}

	watcher, err := watchPods()
	if err != nil {
		s.logger.Log("failed to watch pods: %s", err)
		return
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				s.logger.Log("reconnecting pods watcher")
				watcher.Stop()
				watcher, err = watchPods()
				if err != nil {
					s.logger.Log("failed to watch pods: %s", err)
					return
				}
				continue
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				// non-pod event, skip it
				continue
			}
			trackPods(pod)
			podsListOptions.ResourceVersion = pod.ResourceVersion
		}
	}
}

func (s *Streamer) streamPod(stop <-chan struct{}, podName string, buf chan<- LogEntry) {
	s.logger.Log("streaming pod: %s", podName)
	defer s.logger.Log("pod stream has stopped: %s", podName)

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	podLogOptions := s.podLogOptions.DeepCopy()
	podLogOptions.Follow = s.follow
	podLogOptions.Timestamps = true
	stream, err := s.podsClient.GetLogs(podName, podLogOptions).Stream(streamCtx)
	if err != nil {
		s.logger.Log("failed to start log stream for pod %s: %s", podName, err)
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case <-stop:
			return
		default:
		}

		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		content := parts[1]
		timestamp, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			s.logger.Log("unable to decode log timestamp: %s", err)
			// The current timestamp is the next best substitute. This won't be shown, but will be used
			// for sorting
			timestamp = time.Now()
			content = line
		}
		if s.logFilter == nil || s.logFilter.FilterLog(content) {
			buf <- LogEntry{Time: timestamp, Log: content}
		}
	}
}

func (s *Streamer) consumeLogs(ctx context.Context, buf <-chan LogEntry) {
	ticker := time.NewTicker(s.emitLogsInterval)
	defer ticker.Stop()

	var unsorted logEntries

	sortThenSend := func() {
		if len(unsorted) < 1 {
			return
		}

		sort.Sort(unsorted)
		s.logsConsumer.OnLogs(unsorted)
		unsorted = nil
	}

	// make sure all saved logs are emitted
	defer sortThenSend()

	for {
		select {
		case <-ctx.Done():
			return
		case logEntry, ok := <-buf:
			if !ok {
				return
			}

			unsorted = append(unsorted, logEntry)
		case <-ticker.C:
			sortThenSend()
		}
	}
}
