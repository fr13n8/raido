package netstack

import (
	"errors"

	"github.com/rs/zerolog/log"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

func ttlOption(ttl uint8) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.DefaultTTLOption(ttl)
		if tcpipErr := s.SetNetworkProtocolOption(ipv4.ProtocolNumber, &opt); tcpipErr != nil {
			log.Error().Msgf("could not set ttl for ipv4: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}
		if tcpipErr := s.SetNetworkProtocolOption(ipv6.ProtocolNumber, &opt); tcpipErr != nil {
			log.Error().Msgf("could not set ttl for ipv6: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}

		return nil
	}
}
