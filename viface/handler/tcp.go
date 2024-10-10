package handler

import (
	"context"
	"fmt"
	"net"

	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/relay"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

func TCP(ctx context.Context, conn quic.Connection, fr *tcp.ForwarderRequest) {
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

	// Open a QUIC stream to communicate with the target.
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not open QUIC stream with target")
		return
	}

	// Determine if the connection is IPv4 or IPv6.
	network := protocol.Networkv4
	if s.LocalAddress.To4() == (tcpip.Address{}) {
		network = protocol.Networkv6
	}

	// Create a structure with IP address and port information.
	ipStruct := protocol.IPAddressWithPortProtocol{
		IP:       net.ParseIP(s.LocalAddress.String()),
		Port:     s.LocalPort,
		Protocol: protocol.TransportTCP,
		Network:  network,
	}

	// Encode the IP and port structure.
	encodedIP, err := ipStruct.Encode()
	if err != nil {
		log.Error().Err(err).Msg("could not encode address")
		return
	}

	// Send the connection establishment request via the QUIC stream.
	encoder := protocol.NewEncoder[protocol.Data](stream)
	if err := encoder.Encode(protocol.Data{
		Command: protocol.EstablishConnectionCmd,
		Body:    encodedIP,
	}); err != nil {
		log.Error().Err(err).Msg("could not send connection establishment data")
		return
	}

	// Receive the response from the QUIC stream.
	decoder := protocol.NewDecoder[protocol.ConnectResponse](stream)
	dec, err := decoder.Decode()
	if err != nil {
		log.Error().Err(err).Msg("could not decode connection establishment response")
		return
	}

	// Check if the connection was established successfully.
	if !dec.Established {
		log.Error().Msgf("failed to establish TCP connection with target: %s",
			net.JoinHostPort(s.LocalAddress.String(), fmt.Sprint(s.LocalPort)))
		return
	}

	// Pipe data between the QUIC stream and the TCP connection.
	relay.Pipe(stream, gonetConn)
}
