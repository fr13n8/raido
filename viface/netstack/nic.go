package netstack

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// Generate unique NIC(Network Interface Controller) id and create.
func createNicOption(ctx context.Context, NICID tcpip.NICID, device stack.LinkEndpoint) Option {
	return func(s *stack.Stack) error {
		if tcperr := s.CreateNICWithOptions(NICID, device, stack.NICOptions{
			// TODO
			// Disabled: false,
			// Context: ctx,
		}); tcperr != nil {
			log.Error().Msgf("create NIC error: %s", tcperr)
			return errors.New(tcperr.String())
		}
		return nil
	}
}

func promiscuousModeOption(NICID tcpip.NICID, v bool) Option {
	return func(s *stack.Stack) error {
		if tcperr := s.SetPromiscuousMode(NICID, v); tcperr != nil {
			log.Debug().Msgf("set promiscuous mode: %s", tcperr)
			return errors.New(tcperr.String())
		}
		return nil
	}
}

func spoofingOption(NICID tcpip.NICID, v bool) Option {
	return func(s *stack.Stack) error {
		if tcperr := s.SetSpoofing(NICID, v); tcperr != nil {
			log.Debug().Msgf("set spoofing: %s", tcperr)
			return errors.New(tcperr.String())
		}
		return nil
	}
}

func forwardingOption(v bool) Option {
	return func(s *stack.Stack) error {
		if tcperr := s.SetForwardingDefaultAndAllNICs(ipv4.ProtocolNumber, v); tcperr != nil {
			log.Error().Msgf("set ipv4 forwarding error: %s", tcperr)
			return errors.New(tcperr.String())
		}
		if tcperr := s.SetForwardingDefaultAndAllNICs(ipv6.ProtocolNumber, v); tcperr != nil {
			log.Error().Msgf("set ipv6 forwarding error: %s", tcperr)
			return errors.New(tcperr.String())
		}
		return nil
	}
}
