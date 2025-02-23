package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/fr13n8/raido/agent"
	"github.com/fr13n8/raido/config"
	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/transport"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	listener     transport.StreamListener
	agentManager *agent.Manager
	connCh       chan transport.StreamConn
	workerPool   *WorkerPool
}

func NewServer(ctx context.Context, tr transport.Transport, address string) (*Server, error) {
	listener, err := tr.Listen(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("could not listen on address: %w", err)
	}

	wp := NewWorkerPool(2, 100, 30*time.Second)
	wp.Start()

	return &Server{
		listener:     listener,
		agentManager: agent.NewAgentManager(),
		connCh:       make(chan transport.StreamConn),
		workerPool:   wp,
	}, nil
}

func (s *Server) ShutdownGracefully(ctx context.Context) error {
	log.Info().Msg("shutting down proxy server gracefully...")
	defer close(s.connCh)
	var errs []error

	if err := s.agentManager.Cleanup(); err != nil {
		errs = append(errs, err)
	}

	if err := s.listener.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	s.workerPool.Stop()
	return nil
}

func (s *Server) Listen(ctx context.Context) error {
	log.Info().Str("addr", fmt.Sprintf("%s/%s", s.listener.Addr().Network(), s.listener.Addr().String())).Msg("proxy server started")

	var g errgroup.Group

	g.Go(func() error {
		s.processConnection(ctx)
		return nil
	})

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

	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), config.ShutdownTimeout) // Set a timeout for graceful shutdown
		defer stop()

		if err := s.ShutdownGracefully(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}

		return nil
	})

	return g.Wait()
}

func (s *Server) processConnection(ctx context.Context) {
	for conn := range s.connCh {
		s.workerPool.Submit(func() {
			s.startHandshake(ctx, conn)
		})
	}
}

func (s *Server) startHandshake(ctx context.Context, conn transport.StreamConn) {
	stream, err := conn.OpenStream(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to open stream")
		return
	}
	defer stream.Close()

	encoder := protocol.NewEncoder[protocol.Data](stream)
	decoder := protocol.NewDecoder[protocol.GetRoutesResp](stream)

	if err := encoder.Encode(protocol.Data{
		Command: protocol.GetRoutesReqCmd,
		Body:    nil,
	}); err != nil {
		log.Error().Err(err).Msg("failed to encode data")
		return
	}

	dec, err := decoder.Decode()
	if err != nil {
		log.Error().Err(err).Msg("failed to decode data")
		return
	}

	var routes []string
	for _, route := range dec.Routes {
		ip, _, err := net.ParseCIDR(route)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse route")
			continue
		}
		if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			routes = append(routes, route)
		}
	}

	a := agent.New(dec.Name, conn, routes)
	s.agentManager.AddAgent(a)

	go func() {
		for {
			_, err := conn.AcceptStream(ctx)
			if err != nil {
				var appErr *quic.ApplicationError
				if errors.As(err, &appErr) {
					if appErr.ErrorCode == protocol.ApplicationOK {
						log.Info().Str("agent_id", a.ID).Msg("agent closed connection")

						if err := s.agentManager.RemoveAgent(a.ID); err != nil {
							log.Error().Err(err).Str("agent_id", a.ID).Msg("failed to remove agent")
						}

						return
					}
				}
				log.Error().Err(err).Msg("failed to accept new stream from agent")
				return
			}
		}
	}()
}
