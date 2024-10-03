package quic

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/fr13n8/raido/agent"
	"github.com/fr13n8/raido/config"
	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/lithammer/shortuuid/v4"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	config       *config.ProxyServer
	tlsConfig    *tls.Config
	listener     *quic.Listener
	agentManager *agent.Manager
	connCh       chan quic.Connection
	workerLimit  int
}

func NewServer(conf *config.ProxyServer) (*Server, error) {
	tlsCert, err := tls.LoadX509KeyPair(conf.TLSConfig.CertFile, conf.TLSConfig.KeyFile)
	if err != nil {
		log.Error().Err(err).Msg("failed to load TLS certificates")
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{protocol.Name},
		ServerName:   conf.TLSConfig.ServerName,
	}

	quicListener, err := quic.ListenAddr(conf.Address, tlsConfig, &quic.Config{
		HandshakeIdleTimeout:       5 * time.Second,
		MaxIdleTimeout:             5 * time.Second,
		KeepAlivePeriod:            1 * time.Second,
		MaxIncomingStreams:         1 << 60,
		MaxIncomingUniStreams:      -1,
		DisablePathMTUDiscovery:    false,
		MaxConnectionReceiveWindow: 30 * (1 << 20), // 30 MB
		MaxStreamReceiveWindow:     6 * (1 << 20),  // 6 MB
		// InitialPacketSize:          1252,           //TODO
		Versions: []quic.Version{quic.Version2},
		// Tracer:   NewClientTracer(&log.Logger, 1),
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create QUIC listener")
		return nil, err
	}

	return &Server{
		config:       conf,
		tlsConfig:    tlsConfig,
		listener:     quicListener,
		agentManager: agent.NewAgentManager(),
		connCh:       make(chan quic.Connection),
		workerLimit:  runtime.NumCPU(), // Limit for concurrent goroutines
	}, nil
}

func (s *Server) ShutdownGracefully(ctx context.Context) error {
	log.Info().Msg("shutting down proxy server gracefully...")
	var errs []error

	agents := s.agentManager.GetAgents()
	for _, a := range agents {
		if err := a.CloseTunnel(); err != nil {
			errs = append(errs, err)
		}
		a.Conn.CloseWithError(protocol.ApplicationOK, "server closing down")
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

func (s *Server) Listen(ctx context.Context) error {
	log.Info().Str("addr", fmt.Sprintf("%s/%s", s.listener.Addr().Network(), s.listener.Addr().String())).Msg("proxy server started")

	var g errgroup.Group

	// Connection processing goroutine
	g.Go(func() error {
		return s.ProcessConnection(ctx)
	})

	// Connection accepting goroutine
	g.Go(func() error {
		defer s.listener.Close()

		for {
			if err := ctx.Err(); err != nil {
				log.Info().Msg("stopping listener")
				return nil
			}

			conn, err := s.listener.Accept(ctx)
			if err != nil {
				if errors.Is(err, quic.ErrServerClosed) || errors.Is(err, context.Canceled) {
					log.Info().Msg("listener closed")
					return nil
				}
				log.Error().Err(err).Msg("failed to accept connection")
				continue
			}
			s.connCh <- conn
		}
	})

	// Shutdown listener when context is done
	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second) // Set a timeout for graceful shutdown
		defer stop()

		close(s.connCh)

		if err := s.ShutdownGracefully(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("shutdown error")
			return err
		}

		return nil
	})

	return g.Wait()
}

func (s *Server) ProcessConnection(ctx context.Context) error {
	sem := make(chan struct{}, s.workerLimit) // Semaphore to limit goroutines

	for conn := range s.connCh {
		sem <- struct{}{} // Acquire a semaphore spot
		go func(conn quic.Connection) {
			defer func() { <-sem }() // Release semaphore spot
			s.StartHandshake(ctx, conn)
		}(conn)
	}

	return nil
}

func (s *Server) StartHandshake(ctx context.Context, conn quic.Connection) {
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to open stream")
		return
	}
	defer stream.Close() // Ensure stream is closed

	encoder := protocol.NewEncoder[protocol.Data](stream)
	decoder := protocol.NewDecoder[protocol.GetRoutesResp](stream)

	// Encode and send the request
	if err := encoder.Encode(protocol.Data{
		Command: protocol.GetRoutesReqCmd,
		Body:    nil,
	}); err != nil {
		log.Error().Err(err).Msg("failed to encode data")
		return
	}

	// Decode the response
	dec, err := decoder.Decode()
	if err != nil {
		log.Error().Err(err).Msg("failed to decode data")
		return
	}

	// Parse routes and filter non-loopback IPv4 addresses
	var routes []string
	for _, route := range dec.Routes {
		ip, _, err := net.ParseCIDR(route)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse route")
			continue
		}
		if !ip.IsLoopback() && ip.To4() != nil {
			routes = append(routes, route)
		}
	}

	// Add the new agent to the agent manager
	s.agentManager.AddAgent(shortuuid.New(), agent.New(dec.Name, conn, routes))
}
