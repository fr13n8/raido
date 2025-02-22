package netstack

import (
	"context"
	"fmt"

	"github.com/fr13n8/raido/proxy/transport"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type NetStack struct {
	Stack *stack.Stack
	// device stack.LinkEndpoint
}

// NewNetStack creates and configures a new network stack.
func NewNetStack(ctx context.Context, device stack.LinkEndpoint, conn transport.StreamConn) (*NetStack, error) {
	// Initialize the network stack with the necessary protocols.
	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
			icmp.NewProtocol6,
		},
	})

	// Get the next NIC ID from the stack.
	nicID := s.NextNICID()

	// Define the configuration options.
	options := []Option{
		tcpSackEnabledOption(true), // Enable TCP SACK.
		// tcpRecovery(tcpip.TCPRACKLossDetection), // Use RACK loss detection.
		tcpUseSynCookies(false),                // Enable SYN cookies.
		routeTableOption(nicID),                // Configure routing.
		forwardingOption(true),                 // Enable packet forwarding.
		ttlOption(64),                          // Set default TTL to 64.
		tcpSendReceiveBufSize(4 * 1024 * 1024), // Set TCP buffer size.
		tcpBufferSizeAutoTune(true),            // Enable auto-tuning for buffer size.
		icmpHandler(conn),                      // Set up ICMP handler.
		tcpHandler(ctx, conn),                  // Set up TCP handler.
		udpHandler(ctx, conn),                  // Set up UDP handler.
		createNicOption(ctx, nicID, device),    // Create NIC with the specified ID.
		promiscuousModeOption(nicID, true),     // Enable promiscuous mode.
		spoofingOption(nicID, true),            // Enable spoofing.
	}

	// Apply the options and return any errors encountered.
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err) // Return error instead of logging fatally.
		}
	}

	return &NetStack{
		Stack: s,
	}, nil
}

func (ns *NetStack) Close() {
	ns.Stack.Close()
}
