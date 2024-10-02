package service

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/fr13n8/raido/agent"
	"github.com/fr13n8/raido/config"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	pb "github.com/fr13n8/raido/service/proto"
	"github.com/fr13n8/raido/service/proto/servicev1connect"
)

var _ servicev1connect.RaidoServiceHandler = (*ServiceHandler)(nil)

type ServiceHandler struct {
	agentManager *agent.Manager
	servicev1connect.UnimplementedRaidoServiceHandler
}

func (s *ServiceHandler) GetSessions(ctx context.Context, req *connect.Request[pb.GetSessionsRequest]) (*connect.Response[pb.GetSessionsResponse], error) {
	log.Debug().Any("req", req).Msg("GetSessions()")
	agents := s.agentManager.GetAgents()

	sessions := make(map[string]*pb.Session, len(agents))
	for id, a := range agents {
		sessions[id] = &pb.Session{
			Name:   a.Name,
			Routes: a.Routes,
			Status: a.TunStatus,
		}
	}

	return connect.NewResponse(&pb.GetSessionsResponse{
		Sessions: sessions,
	}), nil
}

func (s *ServiceHandler) SessionTunnelStart(ctx context.Context, req *connect.Request[pb.SessionTunnelStartRequest]) (*connect.Response[pb.SessionTunnelStartResponse], error) {
	log.Debug().Any("req", req).Msg("SessionTunnelStart()")

	id := req.Msg.SessionId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Warn().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, errors.New("agent not found")
	}

	if err := a.StartTunnel(context.Background()); err != nil {
		log.Error().Err(err).Msgf("could not start tunnel for \"%s\"", id)
		return nil, err
	}

	return connect.NewResponse(&pb.SessionTunnelStartResponse{}), nil
}

func (s *ServiceHandler) SessionTunnelStop(ctx context.Context, req *connect.Request[pb.SessionTunnelStopRequest]) (*connect.Response[pb.SessionTunnelStopResponse], error) {
	log.Info().Any("req", req).Msg("SessionTunnelStop()")

	id := req.Msg.SessionId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Warn().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, errors.New("agent not found")
	}

	if err := a.CloseTunnel(); err != nil {
		log.Error().Err(err).Msgf("could not stop tunnel for \"%s\"", id)
		return nil, err
	}

	return connect.NewResponse(&pb.SessionTunnelStopResponse{}), nil
}

type Server struct {
	h3srv *http3.Server
}

func NewServer(cfg *config.ServiceServer) (*Server, error) {
	tlsCert, err := tls.LoadX509KeyPair(cfg.TLSConfig.CertFile, cfg.TLSConfig.KeyFile)
	if err != nil {
		log.Error().Err(err).Msg("failed to load TLS cert")
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle(servicev1connect.NewRaidoServiceHandler(&ServiceHandler{
		agentManager: agent.NewAgentManager(),
	}))

	h3srv := &http3.Server{
		Addr:    cfg.Address,
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			NextProtos:   []string{"raido-service"},
			ServerName:   cfg.TLSConfig.ServerName,
		},
	}

	return &Server{h3srv: h3srv}, nil
}

func (s *Server) Run(ctx context.Context) error {
	g := &errgroup.Group{}

	g.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("shutting down service server")
		return s.ShutDownGracefully()
	})

	log.Info().Str("addr", s.h3srv.Addr).Msg("service server started")
	if err := s.h3srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Error().Err(err).Msg("failed to start service server")
		return err
	}

	return g.Wait()
}

func (s *Server) ShutDownGracefully() error {
	if err := s.h3srv.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close service server")
		return err
	}

	return nil
}
