package podstream

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/b4fun/podkit"
	"github.com/b4fun/podkit/internal/logger"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type streamerTestCtx struct {
	streamer       *Streamer
	fakeKubeClient *fake.Clientset

	namespace string
	labels    map[string]string
}

func newBaseStreamerTestCtx(t *testing.T, opts ...Option) *streamerTestCtx {
	const testNamespace = "test"

	client := fake.NewSimpleClientset()

	streamer := &Streamer{
		logsConsumer:     LogEntryConsumers{},
		logger:           logger.NoOp,
		emitLogsInterval: 1 * time.Second,
		labelSelector:    "app=test",
	}
	for _, opt := range opts {
		opt(streamer)
	}
	streamer.podsClient = client.CoreV1().Pods(testNamespace)

	return &streamerTestCtx{
		streamer:       streamer,
		fakeKubeClient: client,
		namespace:      testNamespace,
		labels:         map[string]string{"app": "test"},
	}
}

func TestStreamer_NoFollow(t *testing.T) {
	newStreamerTestCtx := func(t *testing.T, opts ...Option) *streamerTestCtx {
		testCtx := newBaseStreamerTestCtx(t, opts...)
		testCtx.streamer.follow = false
		return testCtx
	}

	t.Run("no pods", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		testCtx := newStreamerTestCtx(t)
		err := testCtx.streamer.start(ctx.Done())
		assert.NoError(t, err)
	})

	t.Run("single pod", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		testCtx := newStreamerTestCtx(
			t,
			WithLogger(podkit.LogFunc(func(msg string, args ...interface{}) {
				fmt.Printf(msg+"\n", args...)
			})),
		)

		testPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testCtx.namespace,
				Name:      "test-pod",
				Labels:    testCtx.labels,
			},
		}

		testCtx.fakeKubeClient.CoreV1().Pods(testCtx.namespace).
			Create(ctx, testPod, metav1.CreateOptions{})

		err := testCtx.streamer.start(ctx.Done())
		assert.NoError(t, err)
	})
}

func TestStreamer_Follow(t *testing.T) {
	newStreamerTestCtx := func(t *testing.T, opts ...Option) *streamerTestCtx {
		testCtx := newBaseStreamerTestCtx(t, opts...)
		testCtx.streamer.follow = true
		return testCtx
	}

	t.Run("no pods", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		testCtx := newStreamerTestCtx(t)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := testCtx.streamer.start(ctx.Done())
			assert.NoError(t, err)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()
	})

	t.Run("single pod", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		testCtx := newStreamerTestCtx(
			t,
			WithLogger(podkit.LogFunc(func(msg string, args ...interface{}) {
				fmt.Printf(msg+"\n", args...)
			})),
		)

		testPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testCtx.namespace,
				Name:      "test-pod",
				Labels:    testCtx.labels,
			},
		}

		testCtx.fakeKubeClient.CoreV1().Pods(testCtx.namespace).
			Create(ctx, testPod, metav1.CreateOptions{})

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := testCtx.streamer.start(ctx.Done())
			assert.NoError(t, err)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()
	})
}
