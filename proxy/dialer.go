package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/relay"
	"github.com/fr13n8/raido/proxy/transport"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
)

type Dialer struct {
	address    string
	streamCh   chan transport.Stream
	tr         transport.Transport
	workerPool *WorkerPool
}

func NewDialer(ctx context.Context, tr transport.Transport, address string) *Dialer {
	wp := NewWorkerPool(2, 100, 30*time.Second)
	wp.Start()

	return &Dialer{
		streamCh: make(chan transport.Stream, runtime.NumCPU()),
		tr:       tr,
		address:  address, workerPool: wp,
	}
}

func (d *Dialer) dialAndServer(ctx context.Context) error {
	log.Info().Msgf("attempting connection to %s", d.address)
	conn, err := d.tr.Dial(ctx, d.address)
	if err != nil {
		return fmt.Errorf("could not dial address: %w", err)
	}

	log.Info().Msgf("starting dialing to %s", d.address)
	var g errgroup.Group

	g.Go(func() error {
		<-ctx.Done()
		close(d.streamCh)

		return conn.CloseWithError(protocol.ApplicationOK, "client closing down")
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("context cancelled")
				return nil
			default:
				stream, err := conn.AcceptStream(ctx)
				if err != nil {
					var appErr *quic.ApplicationError
					if errors.As(err, &appErr) ||
						errors.Is(err, context.Canceled) ||
						errors.Is(err, context.DeadlineExceeded) {
						log.Info().Msg("connection closed")
						break
					}

					log.Error().Err(err).Msg("failed to accept stream")
					continue
				}
				d.streamCh <- stream
			}
		}
	})

	g.Go(func() error {
		d.processConnection(ctx)

		log.Info().Msg("connection processing stopped")
		return nil
	})

	return g.Wait()
}

func (d *Dialer) Run(ctx context.Context) error {
	defer d.workerPool.Stop()

	return wait.ExponentialBackoffWithContext(ctx, DefaultBackoff, func(context.Context) (done bool, err error) {
		if err := d.dialAndServer(ctx); err != nil {
			log.Error().Err(err).Msg("could not dial and serve")
			if errors.Is(err, context.Canceled) {
				return false, nil
			}

			return false, nil
		}

		return true, nil
	})
}

func (d *Dialer) processConnection(ctx context.Context) {
	for stream := range d.streamCh {
		d.workerPool.Submit(func() {
			d.handleStream(ctx, stream)
		})
	}
}

func (d *Dialer) handleStream(ctx context.Context, stream transport.Stream) {
	decoder := protocol.NewDecoder[protocol.Data](stream)
	dec, err := decoder.Decode()
	if err != nil {
		log.Error().Err(err).Msg("could not decode data")
		return
	}

	switch dec.Command {
	case protocol.GetRoutesReqCmd:
		d.handleGetRoutesRequest(stream)
	case protocol.EstablishConnectionCmd:
		d.handleConnectionRequest(ctx, stream, dec)
	default:
		log.Error().Msg("unknown command")
	}
}

func (d *Dialer) handleGetRoutesRequest(stream transport.Stream) {
	addrs, err := GetNetRoutes()
	if err != nil {
		log.Error().Err(err).Msg("could not get network routes")
		return
	}

	encoder := protocol.NewEncoder[protocol.GetRoutesResp](stream)
	if err := encoder.Encode(protocol.GetRoutesResp{
		Name:   GetUserAndHostname(),
		Routes: addrs,
	}); err != nil {
		log.Error().Err(err).Msg("could not encode network routes response")
	}
}

func (d *Dialer) handleConnectionRequest(ctx context.Context, stream transport.Stream, dec protocol.Data) {
	connRequest, err := protocol.Decode(dec.Body)
	if err != nil {
		log.Error().Err(err).Msg("could not decode connection request")
		return
	}

	network := map[bool]string{true: "tcp", false: "udp"}[connRequest.Protocol == protocol.TransportTCP]
	version := map[bool]string{true: "4", false: "6"}[connRequest.Network == protocol.Networkv4]

	encoder := protocol.NewEncoder[protocol.ConnectResponse](stream)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	targetConn, err := (&net.Dialer{}).DialContext(ctx, network+version, net.JoinHostPort(connRequest.IP.String(), fmt.Sprintf("%d", connRequest.Port)))
	if err != nil {
		log.Error().Err(err).Msg("could not dial target")
		if err := encoder.Encode(protocol.ConnectResponse{Established: false}); err != nil {
			log.Error().Err(err).Msg("could not encode connection response")
		}
		return
	}

	if err := encoder.Encode(protocol.ConnectResponse{Established: true}); err != nil {
		log.Error().Err(err).Msg("could not encode connection response")
	}
	go relay.Pipe(targetConn, stream)
}

func GetNetRoutes() ([]string, error) {
	netifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("could not get network interfaces: %w", err)
	}

	var addrs []string
	for _, iface := range netifaces {
		addresses, err := iface.Addrs()
		if err != nil {
			log.Error().Err(err).Msg("could not get network addresses")
			continue
		}
		for _, addr := range addresses {
			addrs = append(addrs, addr.String())
		}
	}

	return addrs, nil
}

func GetUserAndHostname() string {
	hostname, _ := os.Hostname()
	userinfo, err := user.Current()
	if err != nil {
		log.Error().Err(err).Msg("could not get user info")
		return fmt.Sprintf("unknown@%s", hostname)
	}
	return fmt.Sprintf("%s@%s", userinfo.Username, hostname)
}
