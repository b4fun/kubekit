package podforward

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/b4fun/kubekit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// WithLogger sets the logger to be used by the streamer.
func WithLogger(logger kubekit.Logger) Option {
	return func(forwarder *Forwarder) error {
		forwarder.logger = logger
		return nil
	}
}

// FromSelectedPods sets the label selector.
func FromSelectedPods(labelSelector string) Option {
	return func(forwarder *Forwarder) error {
		forwarder.labelSelector = labelSelector
		return nil
	}
}

// FromService sets the label selector by service name.
// It expects the service exists before the forwarder is created.
func FromService(serviceName string) Option {
	return func(forwarder *Forwarder) error {
		if forwarder.client == nil {
			return errors.New("no kube client provided")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		svc, err := forwarder.client.CoreV1().Services(forwarder.namespace).
			Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get service %q: %w", serviceName, err)
		}

		forwarder.labelSelector = labels.SelectorFromSet(svc.Spec.Selector).String()

		return nil
	}
}

type portPairOptionBuilder struct {
	portPair
}

func (p *portPairOptionBuilder) complete() Option {
	return func(forwarder *Forwarder) error {
		if p.remotePort == PortUnspecified {
			return fmt.Errorf("remote port is reuqired")
		}

		forwarder.ports = append(forwarder.ports, p.portPair)

		return nil
	}
}

func (p *portPairOptionBuilder) FromRemotePort(remotePort uint16) Option {
	p.remotePort = remotePort

	return p.complete()
}

func (p *portPairOptionBuilder) ToLocalPort(localPort uint16) Option {
	p.localPort = localPort

	return p.complete()
}

type FromRemotePortOption interface {
	FromRemotePort(remotePort uint16) Option
}

type ToLocalPortOption interface {
	ToLocalPort(localPort uint16) Option
}

// FromRemotePort creates a new option to specify the remote port.
func FromRemotePort(remotePort uint16) ToLocalPortOption {
	return &portPairOptionBuilder{
		portPair{remotePort: remotePort},
	}
}

// ToLocalPort creates a new option to specify the local port.
func ToLocalPort(localPort uint16) FromRemotePortOption {
	return &portPairOptionBuilder{
		portPair{localPort: localPort},
	}
}
