package device

import (
	"context"
	"net"

	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// TUNDevice abstracts a virtual network TUN device.
type TUNDevice interface {
	Name() string
	AddSubnet(context.Context, *net.IPNet) error
	RemoveSubnet(context.Context, *net.IPNet) error
	Dev() stack.LinkEndpoint
}
