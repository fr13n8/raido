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
	"github.com/fr13n8/raido/proxy/transport"
	"github.com/fr13n8/raido/proxy/transport/core"
	"github.com/fr13n8/raido/proxy/transport/quic"
	"github.com/fr13n8/raido/proxy/transport/tcp"
	"github.com/fr13n8/raido/utils/certs"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

var _ serviceconnect.RaidoServiceHandler = (*ServiceHandler)(nil)

type ServiceHandler struct {
	agentManager        *agent.Manager
	proxyServerInstance *core.Server
	ctx                 context.Context
	proxyCancell        context.CancelFunc
	serviceconnect.UnimplementedRaidoServiceHandler
}

func (s *ServiceHandler) ProxyStart(ctx context.Context, req *connect.Request[pb.ProxyStartRequest]) (*connect.Response[pb.ProxyStartResponse], error) {
	log.Info().Any("req", req).Msg("ProxyStart()")

	if s.proxyServerInstance != nil {
		log.Info().Msg("proxy server instance already exists")
		return nil, fmt.Errorf("proxy server already exists")
	}

	proxyAddr := req.Msg.ProxyAddress
	transportProtocol := req.Msg.TransportProtocol

	cm := certs.NewSelfSignedCertManager("raido_proxy", config.RaidoPath)
	tc, err := cm.GetTLSConfig()
	if err != nil {
		log.Error().Err(err).Msg("failed to get tls config")
		return nil, fmt.Errorf("failed to get tls config")
	}
	tc.NextProtos = []string{protocol.Name}

	var transportImpl transport.Transport
	switch transportProtocol {
	case "quic":
		transportImpl = quic.NewQUICTransport(tc)
	case "tcp":
		transportImpl = tcp.NewTCPTransport(tc)
	default:
		return nil, fmt.Errorf("unsupported transport protocol: %s", transportProtocol)
	}

	s.proxyServerInstance, err = core.NewServer(ctx, transportImpl, proxyAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed to create proxy server")
		return nil, fmt.Errorf("failed to create proxy server")
	}

	go func() {
		ctx, cancel := context.WithCancel(s.ctx)
		s.proxyCancell = cancel
		if err := s.proxyServerInstance.Listen(ctx); err != nil {
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

func (s *ServiceHandler) ProxyStop(ctx context.Context, req *connect.Request[pb.Empty]) (*connect.Response[pb.Empty], error) {
	log.Info().Any("req", req).Msg("ProxyStop()")

	if s.proxyServerInstance == nil {
		log.Info().Msg("proxy server instance is nil")
		return connect.NewResponse(&pb.Empty{}), nil
	}

	s.proxyCancell()

	s.proxyServerInstance = nil

	return connect.NewResponse(&pb.Empty{}), nil
}

func (s *ServiceHandler) AgentList(ctx context.Context, req *connect.Request[pb.Empty]) (*connect.Response[pb.AgentListResponse], error) {
	log.Info().Any("req", req).Msg("GetAgents()")
	agentsResponse := s.agentManager.GetAgents()

	agents := make(map[string]*pb.Agent, len(agentsResponse))
	for id, a := range agentsResponse {
		agents[id] = &pb.Agent{
			Name:   a.Name,
			Routes: a.Routes,
		}
	}

	return connect.NewResponse(&pb.AgentListResponse{
		Agents: agents,
	}), nil
}

func (s *ServiceHandler) TunnelStart(ctx context.Context, req *connect.Request[pb.TunnelStartRequest]) (*connect.Response[pb.Empty], error) {
	log.Info().Any("req", req).Msg("AgentTunnelStart()")

	id := req.Msg.AgentId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Info().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, fmt.Errorf("agent with id \"%s\" doesnt exist", id)
	}

	if err := a.StartTunnel(s.ctx, req.Msg.Routes); err != nil {
		log.Error().Err(err).Msgf("failed to start tunnel for \"%s\"", id)
		return nil, fmt.Errorf("failed to start tunnel for \"%s\"", id)
	}

	return connect.NewResponse(&pb.Empty{}), nil
}

func (s *ServiceHandler) TunnelStop(ctx context.Context, req *connect.Request[pb.TunnelStopRequest]) (*connect.Response[pb.Empty], error) {
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

	return connect.NewResponse(&pb.Empty{}), nil
}

func (s *ServiceHandler) TunnelAddRoute(ctx context.Context, req *connect.Request[pb.TunnelAddRouteRequest]) (*connect.Response[pb.Empty], error) {
	log.Info().Any("req", req).Msg("AgentTunnelAddRoute()")

	id := req.Msg.AgentId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Error().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, fmt.Errorf("agent with id \"%s\" doesnt exist", id)
	}

	if a.Tunnel == nil {
		log.Error().Msgf("tunnel for agent \"%s\" is nil", id)
		return nil, fmt.Errorf("tunnel for agent \"%s\" is nil", id)
	}

	if err := a.Tunnel.AddRoutes(req.Msg.Routes...); err != nil {
		log.Error().Err(err).Msgf("failed to add route to tunnel for \"%s\"", id)
		return nil, fmt.Errorf("failed to add route to tunnel for \"%s\"", id)
	}

	return connect.NewResponse(&pb.Empty{}), nil
}

func (s *ServiceHandler) TunnelRemoveRoute(ctx context.Context, req *connect.Request[pb.TunnelRemoveRouteRequest]) (*connect.Response[pb.Empty], error) {
	log.Info().Any("req", req).Msg("AgentTunnelRemoveRoute()")

	id := req.Msg.AgentId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Error().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, fmt.Errorf("agent with id \"%s\" doesnt exist", id)
	}

	if a.Tunnel == nil {
		log.Error().Msgf("tunnel for agent \"%s\" is nil", id)
		return nil, fmt.Errorf("tunnel for agent \"%s\" is nil", id)
	}

	if err := a.Tunnel.RemoveRoutes(req.Msg.Routes...); err != nil {
		log.Error().Err(err).Msgf("failed to remove route from tunnel for \"%s\"", id)
		return nil, fmt.Errorf("failed to remove route from tunnel for \"%s\"", id)
	}

	return connect.NewResponse(&pb.Empty{}), nil
}

func (s *ServiceHandler) TunnelPause(ctx context.Context, req *connect.Request[pb.TunnelPauseRequest]) (*connect.Response[pb.Empty], error) {
	log.Info().Any("req", req).Msg("AgentTunnelPause()")

	id := req.Msg.AgentId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Error().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, fmt.Errorf("agent with id \"%s\" doesnt exist", id)
	}

	if a.Tunnel == nil {
		log.Error().Msgf("tunnel for agent \"%s\" is nil", id)
		return nil, fmt.Errorf("tunnel for agent \"%s\" is nil", id)
	}

	if err := a.Tunnel.Pause(); err != nil {
		log.Error().Err(err).Msgf("failed to pause tunnel for \"%s\"", id)
		return nil, fmt.Errorf("failed to pause tunnel for \"%s\"", id)
	}

	return connect.NewResponse(&pb.Empty{}), nil
}

func (s *ServiceHandler) TunnelResume(ctx context.Context, req *connect.Request[pb.TunnelResumeRequest]) (*connect.Response[pb.Empty], error) {
	log.Info().Any("req", req).Msg("AgentTunnelResume()")

	id := req.Msg.AgentId

	a := s.agentManager.GetAgent(id)
	if a == nil {
		log.Error().Msgf("agent with id \"%s\" doesnt exist", id)
		return nil, fmt.Errorf("agent with id \"%s\" doesnt exist", id)
	}

	if a.Tunnel == nil {
		log.Error().Msgf("tunnel for agent \"%s\" is nil", id)
		return nil, fmt.Errorf("tunnel for agent \"%s\" is nil", id)
	}

	if err := a.Tunnel.Resume(); err != nil {
		log.Error().Err(err).Msgf("failed to resume tunnel for \"%s\"", id)
		return nil, fmt.Errorf("failed to resume tunnel for \"%s\"", id)
	}

	return connect.NewResponse(&pb.Empty{}), nil
}

func (s *ServiceHandler) TunnelList(ctx context.Context, req *connect.Request[pb.Empty]) (*connect.Response[pb.TunnelListResponse], error) {
	log.Info().Any("req", req).Msg("AgentTunnelList()")

	tunnels := make([]*pb.Tunnel, 0, len(s.agentManager.GetAgents()))
	for id, a := range s.agentManager.GetAgents() {
		if a.Tunnel == nil {
			log.Info().Msgf("tunnel for agent \"%s\" is nil", id)
			continue
		}

		routes, err := a.Tunnel.ActiveRoutes()
		if err != nil {
			log.Error().Err(err).Msgf("failed to get routes for \"%s\"", id)
			continue
		}

		addr, err := a.Tunnel.GetLoopbackRoute()
		if err != nil {
			log.Error().Err(err).Msgf("failed to get address for \"%s\"", id)
		}

		tunnels = append(tunnels, &pb.Tunnel{
			Routes:    routes,
			Status:    a.Tunnel.Status(),
			AgentId:   id,
			Interface: a.Tunnel.Name(),
			Loopback:  addr,
		})
	}

	return connect.NewResponse(&pb.TunnelListResponse{
		Tunnels: tunnels,
	}), nil
}

func (s *ServiceHandler) AgentRemove(ctx context.Context, req *connect.Request[pb.AgentRemoveRequest]) (*connect.Response[pb.Empty], error) {
	log.Info().Any("req", req).Msg("AgentRemove()")

	id := req.Msg.AgentId
	a := s.agentManager.GetAgent(id)

	s.agentManager.RemoveAgent(id)

	a.Conn.CloseWithError(protocol.ApplicationOK, "server closing down")
	if err := a.CloseTunnel(); err != nil {
		log.Error().Err(err).Msgf("failed to close tunnel for \"%s\"", id)
		return nil, fmt.Errorf("failed to close tunnel for \"%s", id)
	}

	return connect.NewResponse(&pb.Empty{}), nil
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
