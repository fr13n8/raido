package tun

import (
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// TUNDevice abstracts a virtual network TUN device.
type TUNDevice interface {
	Name() string
	Dev() stack.LinkEndpoint
}
