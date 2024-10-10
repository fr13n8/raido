package netstack

import (
	"context"
	"errors"
	"fmt"

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
			return fmt.Errorf("create NIC error: %w", errors.New(tcperr.String()))
		}
		return nil
	}
}

func promiscuousModeOption(NICID tcpip.NICID, v bool) Option {
	return func(s *stack.Stack) error {
		if tcperr := s.SetPromiscuousMode(NICID, v); tcperr != nil {
			return fmt.Errorf("set promiscuous mode error: %w", errors.New(tcperr.String()))
		}
		return nil
	}
}

func spoofingOption(NICID tcpip.NICID, v bool) Option {
	return func(s *stack.Stack) error {
		if tcperr := s.SetSpoofing(NICID, v); tcperr != nil {
			return fmt.Errorf("set spoofing error: %w", errors.New(tcperr.String()))
		}
		return nil
	}
}

func forwardingOption(v bool) Option {
	return func(s *stack.Stack) error {
		if tcperr := s.SetForwardingDefaultAndAllNICs(ipv4.ProtocolNumber, v); tcperr != nil {
			return fmt.Errorf("set ipv4 forwarding error: %w", errors.New(tcperr.String()))
		}
		if tcperr := s.SetForwardingDefaultAndAllNICs(ipv6.ProtocolNumber, v); tcperr != nil {
			return fmt.Errorf("set ipv6 forwarding error: %w", errors.New(tcperr.String()))
		}
		return nil
	}
}
