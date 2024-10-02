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
	"time"

	"github.com/fr13n8/raido/config"
	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/relay"
	"golang.org/x/sync/errgroup"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
)

type Dialer struct {
	TLSConfig   *tls.Config
	quicAddress string
	conn        quic.Connection
	streamCh    chan quic.Stream
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

	log.Info().Msgf("attempting connection to %s", conf.ProxyAddress)
	conn, err := quic.DialAddr(ctx, conf.ProxyAddress, tlsConf, &quic.Config{
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
	})
	if err != nil {
		log.Error().Err(err).Msg("could not dial QUIC")
		return nil, err
	}

	return &Dialer{
		TLSConfig:   tlsConf,
		quicAddress: conf.ProxyAddress,
		conn:        conn,
		streamCh:    make(chan quic.Stream, 10), // Buffered channel to avoid blocking
	}, nil
}

func (d *Dialer) DialAndServe(ctx context.Context) error {
	log.Info().Msgf("starting dialing to %s", d.quicAddress)
	var g errgroup.Group

	// Handle context cancellation and connection closing
	g.Go(func() error {
		<-ctx.Done()
		close(d.streamCh)

		d.conn.CloseWithError(protocol.ApplicationOK, "client closing down")

		return nil
	})

	// Process QUIC streams
	g.Go(func() error {
		for {
			stream, err := d.conn.AcceptStream(ctx)
			if err != nil {
				var appErr *quic.ApplicationError
				if errors.As(err, &appErr) || errors.Is(err, context.Canceled) {
					log.Info().Msg("connection closed")
					return nil
				}

				log.Error().Err(err).Msg("failed to accept QUIC stream")
				return err
			}
			d.streamCh <- stream
		}
	})

	// Start worker pool to process QUIC streams
	g.Go(func() error {
		if err := d.ProcessConnection(ctx); err != nil {
			log.Error().Err(err).Msg("could not process connection")
		}

		log.Info().Msg("connection processing stopped")
		return nil
	})

	return g.Wait()
}

func (d *Dialer) ProcessConnection(ctx context.Context) error {
	// workerCount := runtime.NumCPU()
	// sem := make(chan struct{}, workerCount)

	for stream := range d.streamCh {
		// sem <- struct{}{} // Acquire a worker slot
		go func(s quic.Stream) {
			// defer func() { <-sem }()   // Release worker slot when done
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
