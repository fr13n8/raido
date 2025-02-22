package netstack

import (
	"context"

	"github.com/fr13n8/raido/proxy/transport"
	"github.com/fr13n8/raido/viface/handler"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

func udpHandler(ctx context.Context, conn transport.StreamConn) Option {
	return func(s *stack.Stack) error {
		udpForwarder := udp.NewForwarder(s, func(fr *udp.ForwarderRequest) {
			go handler.UDP(ctx, conn, fr)
		})
		s.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)
		return nil
	}
}
