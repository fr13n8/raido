package netstack

import (
	"context"
	"errors"

	"github.com/fr13n8/raido/viface/handler"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
)

func tcpSackEnabledOption(v bool) Option {
	return func(s *stack.Stack) error {
		sackEnabledOpt := tcpip.TCPSACKEnabled(v)
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &sackEnabledOpt); tcpipErr != nil {
			log.Error().Msgf("could not enable TCP SACK: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}
		return nil
	}
}

func tcpRecovery(v tcpip.TCPRecovery) Option {
	return func(s *stack.Stack) error {
		rackEnabledOpt := v
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &rackEnabledOpt); tcpipErr != nil {
			log.Error().Msgf("could not enable TCP RACK: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}
		return nil
	}
}

func tcpUseSynCookies(v bool) Option {
	return func(s *stack.Stack) error {
		synCookies := tcpip.TCPAlwaysUseSynCookies(v)
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &synCookies); tcpipErr != nil {
			log.Error().Msgf("could not disable TCP SYN COOKIES: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}
		return nil
	}
}

func tcpHandler(ctx context.Context, conn quic.Connection) Option {
	return func(s *stack.Stack) error {
		// Set the TCP forwarder with a larger backlog size to handle more concurrent connections.
		tcpForwarder := tcp.NewForwarder(s, 0, 1024, func(fr *tcp.ForwarderRequest) {
			go handler.TCP(ctx, conn, fr)
		})
		s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)
		return nil
	}
}

func tcpBufferSizeAutoTune(v bool) Option {
	return func(s *stack.Stack) error {
		opt := tcpip.TCPModerateReceiveBufferOption(v)
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt); tcpipErr != nil {
			log.Error().Msgf("could not enable receive buffer auto-tunning: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}
		return nil
	}
}

func tcpSendReceiveBufSize(size int) Option {
	return func(s *stack.Stack) error {
		rOpt := tcpip.TCPReceiveBufferSizeRangeOption{Min: 1, Default: size, Max: size}
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &rOpt); tcpipErr != nil {
			log.Error().Msgf("could not set TCP receive buffer size: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}

		sOpt := tcpip.TCPSendBufferSizeRangeOption{Min: 1, Default: size, Max: size}
		if tcpipErr := s.SetTransportProtocolOption(tcp.ProtocolNumber, &sOpt); tcpipErr != nil {
			log.Error().Msgf("could not set TCP send buffer size: %s", tcpipErr)
			return errors.New(tcpipErr.String())
		}

		return nil
	}
}
