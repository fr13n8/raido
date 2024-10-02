package netstack

import "gvisor.dev/gvisor/pkg/tcpip/stack"

type Option func(*stack.Stack) error
