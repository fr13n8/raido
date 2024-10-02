package netstack

import (
	"github.com/fr13n8/raido/viface/handler"
	"github.com/quic-go/quic-go"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

func icmpHandler(conn quic.Connection) Option {
	return func(s *stack.Stack) error {
		return handler.StartResponder(s, conn)
	}
}
