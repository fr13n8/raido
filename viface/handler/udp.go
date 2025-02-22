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
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

func UDP(ctx context.Context, conn transport.StreamConn, fr *udp.ForwarderRequest) {
	// Create endpoint as quickly as possible to avoid UDP
	// race conditions, when user sends multiple frames
	// one after another.
	var wq waiter.Queue
	ep, tcperr := fr.CreateEndpoint(&wq)
	if tcperr != nil {
		log.Error().Msgf("could not create UDP endpoint: %s", tcperr)
		return
	}

	// Set up the UDP connection with the new endpoint and pipe data
	gonetConn := gonet.NewUDPConn(&wq, ep)

	// Identify the flow and log it for better visibility
	s := fr.ID()
	log.Info().Msgf("received UDP flow from %s to %s",
		net.JoinHostPort(s.RemoteAddress.String(), fmt.Sprint(s.RemotePort)),
		net.JoinHostPort(s.LocalAddress.String(), fmt.Sprint(s.LocalPort)))

	// Open the QUIC stream asynchronously to avoid blocking
	stream, err := conn.OpenStream(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not open QUIC stream with target")
		return
	}

	// Handle protocol versioning and IP conversion
	network := protocol.Networkv4
	if s.LocalAddress.To4() == (tcpip.Address{}) {
		network = protocol.Networkv6
	}

	// If the address is from a reserved network range, forward to 127.0.0.1
	localAddress := net.ParseIP(s.LocalAddress.String())
	if ip.LoopbackRoute.Network.Contains(localAddress) {
		localAddress = net.ParseIP("127.0.0.1")
	}

	// Prepare the IP and port encoding for the protocol
	ipStruct := protocol.IPAddressWithPortProtocol{
		IP:       localAddress,
		Port:     s.LocalPort,
		Protocol: protocol.TransportUDP,
		Network:  network,
	}

	// Encode the address and send to the stream
	encodedIP, err := ipStruct.Encode()
	if err != nil {
		log.Error().Err(err).Msg("could not encode IP address")
		return
	}

	encoder := protocol.NewEncoder[protocol.Data](stream)
	decoder := protocol.NewDecoder[protocol.ConnectResponse](stream)

	// Send the connection establishment command
	if err := encoder.Encode(protocol.Data{
		Command: protocol.EstablishConnectionCmd,
		Body:    encodedIP,
	}); err != nil {
		log.Error().Err(err).Msg("could not send establish connection command")
		return
	}

	// Await the response from the target
	dec, err := decoder.Decode()
	if err != nil {
		log.Error().Err(err).Msg("could not decode response from target")
		return
	}

	// Check if the connection was established successfully
	if !dec.Established {
		log.Error().Msgf("could not establish connection with target UDP:%s",
			net.JoinHostPort(s.LocalAddress.String(), fmt.Sprint(s.LocalPort)))
		return
	}

	// Pipe data between the QUIC stream and the UDP connection.
	relay.Pipe(stream, gonetConn)
}
