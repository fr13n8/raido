package quic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/fr13n8/raido/config"
	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/relay"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
)

var (
	// DefaultBackoff is the default backoff used when dialing and serving
	// a connection.
	DefaultBackoff = wait.Backoff{
		Steps:    5,
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}

	qConf = &quic.Config{
		HandshakeIdleTimeout:       5 * time.Second,
		MaxIdleTimeout:             5 * time.Second,
		KeepAlivePeriod:            1 * time.Second,
		MaxIncomingStreams:         1 << 60,
		MaxIncomingUniStreams:      -1,
		DisablePathMTUDiscovery:    false,
		MaxConnectionReceiveWindow: 30 * (1 << 20), // 30 MB
		MaxStreamReceiveWindow:     6 * (1 << 20),  // 6 MB
		// InitialPacketSize:          1252,
		Versions: []quic.Version{quic.Version2},
		// Tracer:   NewClientTracer(&log.Logger, 1),
	}
)

type Dialer struct {
	quicAddress string
	streamCh    chan quic.Stream
	qConf       *quic.Config
	tlsConf     *tls.Config
}

func NewDialer(ctx context.Context, conf *config.Dialer) (*Dialer, error) {
	tlsConf := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{protocol.Name},
		ServerName:         conf.TLSConfig.ServerName,
		InsecureSkipVerify: conf.TLSConfig.InsecureSkipVerify,
	}

	// Initialize root CA pool
	tlsConf.RootCAs, _ = x509.SystemCertPool()
	if tlsConf.RootCAs == nil {
		tlsConf.RootCAs = x509.NewCertPool()
	}

	if conf.TLSConfig.CertFile != "" {
		caCertRaw, err := os.ReadFile(conf.TLSConfig.CertFile)
		if err != nil {
			log.Error().Err(err).Msg("failed to read cert")
			return nil, err
		}
		if !tlsConf.RootCAs.AppendCertsFromPEM(caCertRaw) {
			log.Error().Err(err).Msg("failed to append cert at path")
			return nil, err
		}
	}

	return &Dialer{
		quicAddress: conf.ProxyAddress,
		streamCh:    make(chan quic.Stream, runtime.NumCPU()), // Buffered channel to avoid blocking
		qConf:       qConf,
		tlsConf:     tlsConf,
	}, nil
}

func (d *Dialer) dialAndServer(ctx context.Context) error {
	log.Info().Msgf("attempting connection to %s", d.quicAddress)
	conn, err := quic.DialAddr(ctx, d.quicAddress, d.tlsConf, d.qConf)
	if err != nil {
		return err
	}

	log.Info().Msgf("starting dialing to %s", d.quicAddress)
	var g errgroup.Group

	ctx, stop := context.WithCancel(ctx)
	// Handle context cancellation and connection closing
	g.Go(func() error {
		<-ctx.Done()
		close(d.streamCh)

		conn.CloseWithError(protocol.ApplicationOK, "client closing down")

		return nil
	})

	// Process QUIC streams
	g.Go(func() error {
		for {
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				var appErr *quic.ApplicationError
				if errors.As(err, &appErr) || errors.Is(err, context.Canceled) {
					log.Info().Msg("connection closed")
					break
				}

				log.Error().Err(err).Msg("failed to accept QUIC stream")
				continue
			}
			d.streamCh <- stream
		}

		stop()
		return nil
	})

	// Start worker pool to process QUIC streams
	g.Go(func() error {
		if err := d.ProcessConnection(ctx); err != nil {
			log.Error().Err(err).Msg("could not process connection")
			return err
		}

		log.Info().Msg("connection processing stopped")
		return nil
	})

	return g.Wait()
}

func (d *Dialer) Run(ctx context.Context) error {
	return wait.ExponentialBackoffWithContext(ctx, DefaultBackoff, func(context.Context) (done bool, err error) {
		if err := d.dialAndServer(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return false, nil
			}

			log.Error().Err(err).Msg("error while attempting to dial server")
			return false, nil
		}

		return true, nil
	})
}

func (d *Dialer) ProcessConnection(ctx context.Context) error {
	workerCount := runtime.NumCPU()
	sem := make(chan struct{}, workerCount)

	for stream := range d.streamCh {
		sem <- struct{}{} // Acquire a worker slot
		go func(s quic.Stream) {
			defer func() { <-sem }()   // Release worker slot when done
			d.handleQUICStream(ctx, s) // Process the stream
		}(stream)
	}

	return nil
}

func (d *Dialer) handleQUICStream(ctx context.Context, stream quic.Stream) {
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

func (d *Dialer) handleGetRoutesRequest(stream quic.Stream) {
	addrs, err := GetNetRoutes()
	if err != nil {
		log.Error().Err(err).Msg("could not get network routes")
		return
	}

	encoder := protocol.NewEncoder[protocol.GetRoutesResp](stream)
	err = encoder.Encode(protocol.GetRoutesResp{
		Name:   GetUserAndHostname(),
		Routes: addrs,
	})
	if err != nil {
		log.Error().Err(err).Msg("could not encode network routes response")
	}
}

func (d *Dialer) handleConnectionRequest(ctx context.Context, stream quic.Stream, dec protocol.Data) {
	connRequest, err := protocol.Decode(dec.Body)
	if err != nil {
		log.Error().Err(err).Msg("could not decode connection request")
		return
	}

	network := map[bool]string{true: "tcp", false: "udp"}[connRequest.Protocol == protocol.TransportTCP]
	version := map[bool]string{true: "4", false: "6"}[connRequest.Network == protocol.Networkv4]

	encoder := protocol.NewEncoder[protocol.ConnectResponse](stream)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	relay.Pipe(targetConn, stream)
}

func GetNetRoutes() ([]string, error) {
	netifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("get interfaces error: %w", err)
	}

	var addrs []string
	for _, iface := range netifaces {
		addresses, err := iface.Addrs()
		if err != nil {
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
		return fmt.Sprintf("unknown@%s", hostname)
	}
	return fmt.Sprintf("%s@%s", userinfo.Username, hostname)
}
