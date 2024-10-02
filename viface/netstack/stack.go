package netstack

import (
	"context"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

// NewNetStack creates and configures a new network stack.
func NewNetStack(ctx context.Context, device stack.LinkEndpoint, conn quic.Connection) (*stack.Stack, error) {
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
	// nicID := s.NextNICID()
	nicID := tcpip.NICID(1)

	// Define the configuration options.
	options := []Option{
		tcpSackEnabledOption(true), // Enable TCP SACK.
		// tcpRecovery(tcpip.TCPRACKLossDetection), // Use RACK loss detection.
		tcpUseSynCookies(false),                // Enable SYN cookies (can affect nmap scans).
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
			log.Error().Err(err).Msg("option failed")
			return nil, err // Return error instead of logging fatally.
		}
	}

	return s, nil
}
