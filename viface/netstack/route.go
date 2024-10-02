package netstack

import (
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

func routeTableOption(NICID tcpip.NICID) Option {
	return func(s *stack.Stack) error {
		s.SetRouteTable([]tcpip.Route{
			{
				Destination: header.IPv4EmptySubnet,
				NIC:         NICID,
			},
			{
				Destination: header.IPv6EmptySubnet,
				NIC:         NICID,
			},
		})
		return nil
	}
}
