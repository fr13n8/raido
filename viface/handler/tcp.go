package handler

import (
	"context"
	"fmt"
	"net"

	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/relay"
	"github.com/fr13n8/raido/proxy/transport"
	"github.com/fr13n8/raido/utils/ip"
	"github.com/rs/zerolog/log"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type TCPHandler struct {
	conn transport.StreamConn
}

func NewTCPHandler(conn transport.StreamConn) *TCPHandler {
	return &TCPHandler{conn: conn}
}

func (h *TCPHandler) HandleRequest(ctx context.Context, fr *tcp.ForwarderRequest) {
	// Create a waiter queue and TCP endpoint for the forwarded connection.
	var wq waiter.Queue
	ep, tcperr := fr.CreateEndpoint(&wq)
	if tcperr != nil {
		log.Error().Msgf("failed to create TCP endpoint: %s", tcperr)
		fr.Complete(true)
		return
	}
	defer fr.Complete(false)

	// Convert the TCP endpoint into a Go net TCP connection.
	gonetConn := gonet.NewTCPConn(&wq, ep)

	// Get the flow info (source and destination addresses and ports).
	s := fr.ID()
	log.Info().Msgf("received TCP flow from %s to %s",
		net.JoinHostPort(s.RemoteAddress.String(), fmt.Sprint(s.RemotePort)),
		net.JoinHostPort(s.LocalAddress.String(), fmt.Sprint(s.LocalPort)))

	// Open a stream to communicate with the target.
	stream, err := h.conn.GetStream(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not open stream with target")
		return
	}
	defer h.conn.PutStream(stream)

	if err := h.establishConnection(ctx, stream, s); err != nil {
		log.Error().Err(err).Msg("Establish connection failed")
		return
	}

	// Pipe data between the stream and the TCP connection.
	if err := relay.Pipe(stream, gonetConn); err != nil {
		log.Error().Err(err).Msg("could not pipe data between stream and TCP connection")
		return
	}
}

func (h *TCPHandler) establishConnection(ctx context.Context, stream transport.Stream, s stack.TransportEndpointID) error {
	// Determine if the connection is IPv4 or IPv6.
	network := protocol.Networkv4
	if s.LocalAddress.To4() == (tcpip.Address{}) {
		network = protocol.Networkv6
	}

	// If the address is from a reserved network range, forward to 127.0.0.1
	localAddress := net.ParseIP(s.LocalAddress.String())
	if ip.LoopbackRoute.Network.Contains(localAddress) {
		localAddress = net.ParseIP("127.0.0.1")
	}

	// Create a structure with IP address and port information.
	ipStruct := protocol.IPAddressWithPortProtocol{
		IP:       localAddress,
		Port:     s.LocalPort,
		Protocol: protocol.TransportTCP,
		Network:  network,
	}

	// Encode the IP and port structure.
	encodedIP, err := ipStruct.Encode()
	if err != nil {
		log.Error().Err(err).Msg("could not encode address")
		return fmt.Errorf("could not encode address: %w", err)
	}

	// Send the connection establishment request via the QUIC stream.
	encoder := protocol.NewEncoder[protocol.Data](stream)
	if err := encoder.Encode(protocol.Data{
		Command: protocol.EstablishConnectionCmd,
		Body:    encodedIP,
	}); err != nil {
		log.Error().Err(err).Msg("could not send connection establishment data")
		return fmt.Errorf("could not send connection establishment data: %w", err)
	}

	// Receive the response from the QUIC stream.
	decoder := protocol.NewDecoder[protocol.ConnectResponse](stream)
	dec, err := decoder.Decode()
	if err != nil {
		log.Error().Err(err).Msg("could not decode connection establishment response")
		return fmt.Errorf("could not decode connection establishment response: %w", err)
	}

	// Check if the connection was established successfully.
	if !dec.Established {
		log.Error().Msgf("failed to establish TCP connection with target: %s",
			net.JoinHostPort(s.LocalAddress.String(), fmt.Sprint(s.LocalPort)))
		return fmt.Errorf("failed to establish TCP connection with target")
	}

	return nil
}
