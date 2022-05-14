package podforward

import (
	"k8s.io/client-go/tools/portforward"
)

type forwardHandle struct {
	portsMapping map[uint16]uint16

	stopFunc func()
}

func newForwardHandle(ports []portforward.ForwardedPort, stopFunc func()) *forwardHandle {
	mapping := make(map[uint16]uint16)
	for _, p := range ports {
		mapping[p.Remote] = p.Local
	}

	return &forwardHandle{
		portsMapping: mapping,
		stopFunc:     stopFunc,
	}
}

var _ ForwardHandle = (*forwardHandle)(nil)

func (h *forwardHandle) LocalPort(remotePort uint16) uint16 {
	v, exists := h.portsMapping[remotePort]
	if !exists {
		return PortUnspecified
	}

	return v
}

func (h *forwardHandle) StopForward() {
	h.stopFunc()
}
