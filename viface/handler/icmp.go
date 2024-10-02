package handler

import (
	"bytes"
	"errors"

	"github.com/quic-go/quic-go"
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

func StartResponder(s *stack.Stack, conn quic.Connection) error {
	var wq waiter.Queue

	ep, err := raw.NewEndpoint(s, ipv4.ProtocolNumber, icmp.ProtocolNumber4, &wq)
	if err != nil {
		log.Debug().Msgf("Could not create raw endpoint: %s", err.String())
		return errors.New(err.String())
	}
	if err := ep.Bind(tcpip.FullAddress{}); err != nil {
		log.Debug().Msgf("Could not bind raw endpoint: %s", err.String())
		return errors.New(err.String())
	}

	// ep, err := s.GetStack().NewRawEndpoint(icmp.ProtocolNumber4, ipv4.ProtocolNumber, &wq, true)
	// if err != nil {
	// 	log.Debug().Msgf("Icmp responder error: %v", err)
	// 	return errors.New(err.String())
	// }

	waitEntry, ch := waiter.NewChannelEntry(waiter.ReadableEvents)
	wq.EventRegister(&waitEntry)

	go func() {
		for {
			var buf bytes.Buffer
			_, err := ep.Read(&buf, tcpip.ReadOptions{})
			if err != nil {
				if _, ok := err.(*tcpip.ErrWouldBlock); !ok {
					log.Debug().Msgf("icmp responder error: %v", err)
				}
				continue
			}

			neth := header.IPv4(buf.Bytes())

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
				continue
			}
			h := header.ICMPv4(v)
			log.Debug().Msgf("received icmp[type=%d code=%d] flow from %s to %s", h.Type(), h.Code(), neth.SourceAddress().String(), neth.DestinationAddress())

			// if h.Type() == header.ICMPv4Echo {
			// 	iph := header.IPv4(packetbuff.NetworkHeader().Slice())
			// 	// Parse network header for destination address.
			// 	dest := iph.DestinationAddress().String()

			// 	// log.Debug().Any("payload", string(neth.Payload())).Send()
			// 	log.Debug().Str("dest", dest).Send()

			// 	// requestPing := goicmp.Echo{
			// 	// 	Seq:  rand.Intn(1 << 16),
			// 	// 	Data: []byte("gopher burrow"),
			// 	// }
			// 	// icmpBytes, _ := (&goicmp.Message{Type: goipv4.ICMPTypeEcho, Code: 0, Body: &requestPing}).Marshal(nil)
			// 	// _ = icmpBytes
			// 	// log.Info().Msg(string(icmpBytes))
			// }

			<-ch
		}

	}()

	return nil
}
