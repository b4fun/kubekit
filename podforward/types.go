package podforward

import (
	"errors"
)

// PortUnspecified is the fallback value of the port.
// THe forwarder generates a random port if this value is used.
const PortUnspecified uint16 = 0

var ErrEmptyPodsListed = errors.New("empty pods listed")

// Option specifies options for configuring the podstream reader.
type Option func(forwarder *Forwarder) error

// ForwardHandle controls access to a pod forwarder.
type ForwardHandle interface {
	// LocalPort returns the local port of the forwarder by remote port.
	LocalPort(remortPort uint16) uint16

	// StopForward stops the forwarder.
	StopForward()
}
