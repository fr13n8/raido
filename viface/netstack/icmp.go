package netstack

import (
	"github.com/fr13n8/raido/proxy/transport"
	"github.com/fr13n8/raido/viface/handler"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

func icmpHandler(conn transport.StreamConn) Option {
	return func(s *stack.Stack) error {
		return handler.StartResponder(s, conn)
	}
}
