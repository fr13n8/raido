package handler

import (
	"bytes"
	"context"
	"errors"

	"github.com/fr13n8/raido/proxy/transport"
	"github.com/rs/zerolog/log"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/raw"
	"gvisor.dev/gvisor/pkg/waiter"
)

type ICMPHandler struct {
	stack *stack.Stack
	conn  transport.StreamConn
}

func NewICMPHandler(s *stack.Stack, conn transport.StreamConn) *ICMPHandler {
	return &ICMPHandler{stack: s, conn: conn}
}

func (h *ICMPHandler) Start(ctx context.Context) error {
	var wq waiter.Queue

	ep, err := raw.NewEndpoint(h.stack, ipv4.ProtocolNumber, icmp.ProtocolNumber4, &wq)
	if err != nil {
		log.Error().Msgf("Could not create raw endpoint: %s", err.String())
		return errors.New(err.String())
	}
	if err := ep.Bind(tcpip.FullAddress{}); err != nil {
		log.Error().Msgf("Could not bind raw endpoint: %s", err.String())
		return errors.New(err.String())
	}

	// ep, err := s.GetStack().NewRawEndpoint(icmp.ProtocolNumber4, ipv4.ProtocolNumber, &wq, true)
	// if err != nil {
	// 	log.Debug().Msgf("Icmp responder error: %v", err)
	// 	return errors.New(err.String())
	// }

	waitEntry, ch := waiter.NewChannelEntry(waiter.ReadableEvents)
	wq.EventRegister(&waitEntry)

	go h.processICMPPackets(ctx, ep, ch)

	return nil
}

func (h *ICMPHandler) processICMPPackets(ctx context.Context, ep tcpip.Endpoint, ch chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			var buf bytes.Buffer
			_, err := ep.Read(&buf, tcpip.ReadOptions{})
			if err != nil {
				if _, ok := err.(*tcpip.ErrWouldBlock); !ok {
					log.Error().Err(errors.New(err.String())).Msg("ICMP read error")
				}
				continue
			}

			neth := header.IPv4(buf.Bytes())
			h.handleICMPPacket(neth, buf)
		}
	}
}

func (h *ICMPHandler) handleICMPPacket(neth header.IPv4, buf bytes.Buffer) {
	view := buffer.MakeWithData(buf.Bytes())
	packetbuff := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload:            view,
		ReserveHeaderBytes: int(neth.HeaderLength()),
		IsForwardedPacket:  true,
	})

	packetbuff.NetworkProtocolNumber = ipv4.ProtocolNumber
	packetbuff.TransportProtocolNumber = icmp.ProtocolNumber4
	packetbuff.NetworkHeader().Consume(int(neth.HeaderLength()))

	v, ok := packetbuff.Data().PullUp(header.ICMPv4MinimumSize)
	if !ok {
		log.Error().Msg("Failed to pull up ICMP header")
		return
	}
	hr := header.ICMPv4(v)

	log.Info().
		Str("source", neth.SourceAddress().String()).
		Str("destination", neth.DestinationAddress().String()).
		Bytes("type", []byte{byte(hr.Type())}).
		Bytes("code", []byte{byte(hr.Code())}).
		Msg("Received ICMP packet")

	// Add ICMP response handling logic here
}
