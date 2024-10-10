package netstack

import (
	"errors"
	"fmt"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

func ttlOption(ttl uint8) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.DefaultTTLOption(ttl)
		if tcpipErr := s.SetNetworkProtocolOption(ipv4.ProtocolNumber, &opt); tcpipErr != nil {
			return fmt.Errorf("could not set ttl for ipv4: %w", errors.New(tcpipErr.String()))
		}
		if tcpipErr := s.SetNetworkProtocolOption(ipv6.ProtocolNumber, &opt); tcpipErr != nil {
			return fmt.Errorf("could not set ttl for ipv6: %w", errors.New(tcpipErr.String()))
		}

		return nil
	}
}
