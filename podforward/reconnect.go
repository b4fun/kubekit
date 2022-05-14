package podforward

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/b4fun/podkit/internal/logger"
	"k8s.io/client-go/rest"
)

// ForwardWithReconnect creates pod forwarder with automatic reconnection support.
// It bails if fails to create first connection. Errors after first connection
// will trigger forwarder reconnection. No error will be returned from the returned
// forward handle until stopped.
func ForwardWithReconnect(
	logger logger.Logger,
	forwardTimeout time.Duration,
	backoffTicker <-chan time.Time,
	restConfig *rest.Config,
	namespace string,
	options ...Option,
) (ForwardHandle, error) {
	forwardFn := func() (ForwardHandle, error) {
		ctx, cancel := context.WithTimeout(context.Background(), forwardTimeout)
		defer cancel()

		return Forward(ctx, restConfig, namespace, options...)
	}

	currentHandle, err := forwardFn()
	if err != nil {
		err = fmt.Errorf("failed to forward pod: %w", err)
		logger.Log(err.Error())
		return nil, err
	}

	reconnectCtx, cancelReconnect := context.WithCancel(context.Background())

	rv := &replacebleForwardHandle{
		stop:    cancelReconnect,
		errChan: make(chan error),
	}

	go func() {
		defer logger.Log("forward stopped")

		for {
			select {
			case <-reconnectCtx.Done():
				// stop reconnection
				return
			case err := <-currentHandle.ErrChan():
				// previous handle stopped, reconnect
				if err != nil {
					logger.Log("reconnecting due to error: %s", err)
				} else {
					logger.Log("reconnecting due to previous handle stopped")
				}
			RECONNECT:
				for {
					select {
					case <-reconnectCtx.Done():
						// stop reconnection
						return
					case <-backoffTicker:
						handle, err := forwardFn()
						if err != nil {
							logger.Log("failed to forward to pod: %s", err)
							continue
						}
						currentHandle = handle
						rv.replace(currentHandle)
						break RECONNECT
					}
				}
			}

		}
	}()

	return rv, nil
}

type replacebleForwardHandle struct {
	stopOnce sync.Once
	stop     func()
	errChan  chan error

	l             sync.Mutex
	forwardHandle ForwardHandle
}

var _ ForwardHandle = (*replacebleForwardHandle)(nil)

func (h *replacebleForwardHandle) LocalPort(remotePort uint16) uint16 {
	h.l.Lock()
	defer h.l.Unlock()

	return h.forwardHandle.LocalPort(remotePort)
}

func (h *replacebleForwardHandle) StopForward() {
	h.stopOnce.Do(func() {
		h.l.Lock()
		if h.forwardHandle != nil {
			h.forwardHandle.StopForward()
		}
		h.l.Unlock()

		h.stop()
		close(h.errChan)
	})
}

func (h *replacebleForwardHandle) ErrChan() <-chan error {
	return h.errChan
}

func (h *replacebleForwardHandle) replace(handle ForwardHandle) {
	h.l.Lock()
	defer h.l.Unlock()

	if h.forwardHandle != nil {
		h.forwardHandle.StopForward()
	}

	h.forwardHandle = handle
}
