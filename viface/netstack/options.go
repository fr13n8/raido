package netstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/fr13n8/raido/proxy/transport"
	"github.com/fr13n8/raido/viface/handler"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type Option func(*stack.Stack) error

func icmpHandler(conn transport.StreamConn) Option {
	return func(s *stack.Stack) error {
		return handler.NewICMPHandler(s, conn).Start(context.TODO())
	}
}

func tcpSackEnabledOption(v bool) Option {
	return func(s *stack.Stack) error {
		sackEnabledOpt := tcpip.TCPSACKEnabled(v)
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &sackEnabledOpt); tcpipErr != nil {
			return fmt.Errorf("could not enable TCP SACK: %w", errors.New(tcpipErr.String()))
		}
		return nil
	}
}

func tcpRecovery(v tcpip.TCPRecovery) Option {
	return func(s *stack.Stack) error {
		rackEnabledOpt := v
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &rackEnabledOpt); tcpipErr != nil {
			return fmt.Errorf("could not enable TCP RACK: %w", errors.New(tcpipErr.String()))
		}
		return nil
	}
}

func tcpUseSynCookies(v bool) Option {
	return func(s *stack.Stack) error {
		synCookies := tcpip.TCPAlwaysUseSynCookies(v)
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &synCookies); tcpipErr != nil {
			return fmt.Errorf("could not disable TCP SYN COOKIES: %w", errors.New(tcpipErr.String()))
		}
		return nil
	}
}

func tcpHandler(ctx context.Context, conn transport.StreamConn) Option {
	return func(s *stack.Stack) error {
		// Set the TCP forwarder with a larger backlog size to handle more concurrent connections.
		tcpForwarder := tcp.NewForwarder(s, 0, 1024, func(fr *tcp.ForwarderRequest) {
			go handler.NewTCPHandler(conn).HandleRequest(ctx, fr)
		})
		s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)
		return nil
	}
}

func tcpBufferSizeAutoTune(v bool) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.TCPModerateReceiveBufferOption(v)
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt); tcpipErr != nil {
			return fmt.Errorf("could not enable receive buffer auto-tunning: %w", errors.New(tcpipErr.String()))
		}
		return nil
	}
}

func tcpSendReceiveBufSize(size int) Option {
	return func(s *stack.Stack) error {
		rOpt := tcpip.TCPReceiveBufferSizeRangeOption{Min: 1, Default: size, Max: size}
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &rOpt); tcpipErr != nil {
			return fmt.Errorf("could not set TCP receive buffer size: %w", errors.New(tcpipErr.String()))
		}

		sOpt := tcpip.TCPSendBufferSizeRangeOption{Min: 1, Default: size, Max: size}
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &sOpt); tcpipErr != nil {
			return fmt.Errorf("could not set TCP send buffer size: %w", errors.New(tcpipErr.String()))
		}

		return nil
	}
}

func udpHandler(ctx context.Context, conn transport.StreamConn) Option {
	return func(s *stack.Stack) error {
		udpForwarder := udp.NewForwarder(s, func(fr *udp.ForwarderRequest) {
			go handler.NewUDPHandler(conn).HandleRequest(ctx, fr)
		})
		s.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)
		return nil
	}
}

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
