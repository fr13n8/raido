package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	pb "github.com/fr13n8/raido/proto/service"
	"github.com/fr13n8/raido/proto/service/serviceconnect"

	"connectrpc.com/connect"
	"github.com/fr13n8/raido/agent"
	"github.com/fr13n8/raido/config"
	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/transport/quic"
	"github.com/fr13n8/raido/utils/certs"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

var _ serviceconnect.RaidoServiceHandler = (*ServiceHandler)(nil)

type ServiceHandler struct {
	agentManager        *agent.Manager
	proxyServerInstance *quic.Server
	ctx                 context.Context
	serviceconnect.UnimplementedRaidoServiceHandler
}

func (s *ServiceHandler) ProxyStart(ctx context.Context, req *connect.Request[pb.ProxyStartRequest]) (*connect.Response[pb.ProxyStartResponse], error) {
	log.Info().Any("req", req).Msg("ProxyStart()")

	if s.proxyServerInstance != nil {
		log.Info().Msg("proxy server instance already exists")
		return nil, fmt.Errorf("proxy server already exists")
	}

	proxyAddr := req.Msg.ProxyAddress

	cm := certs.NewSelfSignedCertManager("raido_proxy", config.RaidoPath)
	tc, err := cm.GetTLSConfig()
	if err != nil {
		log.Error().Err(err).Msg("failed to get tls config")
		return nil, fmt.Errorf("failed to get tls config")
	}
	tc.NextProtos = []string{protocol.Name}

	pCfg := &config.ProxyServer{
		Address:   proxyAddr,
		TLSConfig: tc,
	}

	s.proxyServerInstance, err = quic.NewServer(pCfg)
	if err != nil {
		log.Error().Err(err).Msg("failed to create proxy server")
		return nil, fmt.Errorf("failed to create proxy server")
	}

	go func() {
		if err := s.proxyServerInstance.Listen(s.ctx); err != nil {
			log.Error().Err(err).Msg("Failed to listen proxy")
		}
	}()

	certHash, err := cm.GetCertHash()
	if err != nil {
		log.Error().Err(err).Msg("failed to get cert hash")
		return nil, fmt.Errorf("failed to get cert hash")
	}

	return connect.NewResponse(&pb.ProxyStartResponse{
		CertHash: certHash,
	}), nil
}

func (s *ServiceHandler) ProxyStop(ctx context.Context, req *connect.Request[pb.ProxyStopRequest]) (*connect.Response[pb.ProxyStopResponse], error) {
	log.Info().Any("req", req).Msg("ProxyStop()")

	if s.proxyServerInstance == nil {
		log.Info().Msg("proxy server instance is nil")
		return connect.NewResponse(&pb.ProxyStopResponse{}), nil
	}

	if err := s.proxyServerInstance.ShutdownGracefully(ctx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown proxy server")
		return nil, fmt.Errorf("failed to shutdown proxy server")
	}

	s.proxyServerInstance = nil

	return connect.NewResponse(&pb.ProxyStopResponse{}), nil
}

func (s *ServiceHandler) GetAgents(ctx context.Context, req *connect.Request[pb.GetAgentsRequest]) (*connect.Response[pb.GetAgentsResponse], error) {
	log.Info().Any("req", req).Msg("GetAgents()")
	agentsResponse := s.agentManager.GetAgents()

	agents := make(map[string]*pb.Agent, len(agentsResponse))
	for id, a := range agentsResponse {
		agents[id] = &pb.Agent{
			Name:   a.Name,
			Routes: a.Routes,
			Status: a.TunStatus,
		}
	}

	return connect.NewResponse(&pb.GetAgentsResponse{
		Agents: agents,
	}), nil
}

func (s *ServiceHandler) AgentTunnelStart(ctx context.Context, req *connect.Request[pb.AgentTunnelStartRequest]) (*connect.Response[pb.AgentTunnelStartResponse], error) {
	log.Info().Any("req", req).Msg("AgentTunnelStart()")

	id := req.Msg.AgentId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Info().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, fmt.Errorf("agent with id \"%s\" doesnt exist", id)
	}

	if err := a.StartTunnel(context.Background()); err != nil {
		log.Error().Err(err).Msgf("failed to start tunnel for \"%s\"", id)
		return nil, fmt.Errorf("failed to start tunnel for \"%s\"", id)
	}

	return connect.NewResponse(&pb.AgentTunnelStartResponse{}), nil
}

func (s *ServiceHandler) AgentTunnelStop(ctx context.Context, req *connect.Request[pb.AgentTunnelStopRequest]) (*connect.Response[pb.AgentTunnelStopResponse], error) {
	log.Info().Any("req", req).Msg("AgentTunnelStop()")

	id := req.Msg.AgentId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Error().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, fmt.Errorf("agent with id \"%s\" doesnt exist", id)
	}

	if err := a.CloseTunnel(); err != nil {
		log.Error().Err(err).Msgf("could not stop tunnel for \"%s\"", id)
		return nil, fmt.Errorf("could not stop tunnel for \"%s\"", id)
	}

	return connect.NewResponse(&pb.AgentTunnelStopResponse{}), nil
}

type Server struct {
	serverInstance *http.Server
	ctx            context.Context
}

func NewServer(ctx context.Context, cfg *config.ServiceServer) *Server {
	mux := http.NewServeMux()
	mux.Handle(serviceconnect.NewRaidoServiceHandler(&ServiceHandler{
		agentManager: agent.NewAgentManager(),
		ctx:          ctx,
	}))

	srv := &http.Server{
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	return &Server{
		serverInstance: srv,
		ctx:            ctx,
	}
}

func (s *Server) Run(listener net.Listener) error {
	g := &errgroup.Group{}

	g.Go(func() error {
		<-s.ctx.Done()
		log.Info().Msg("shutting down service server")
		return s.ShutDownGracefully()
	})

	g.Go(func() error {
		if err := s.serverInstance.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to start service server: %w", err)
		}

		return nil
	})

	return g.Wait()
}

func (s *Server) ShutDownGracefully() error {
	if err := s.serverInstance.Close(); err != nil {
		return fmt.Errorf("failed to close service server: %w", err)
	}

	return nil
}
