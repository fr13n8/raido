package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fr13n8/raido/proto/service"
	"github.com/fr13n8/raido/proto/service/serviceconnect"

	"connectrpc.com/connect"
	"github.com/fr13n8/raido/config"
	"github.com/peterbourgon/unixtransport"
)

type ClientKey struct{}

type Client struct {
	serviceClient serviceconnect.RaidoServiceClient
}

func NewClient(ctx context.Context, cfg *config.ServiceDialer) *Client {
	roundTripper := &http.Transport{
		ForceAttemptHTTP2: true,
	}
	split := strings.Split(cfg.ServiceAddress, "://")
	serviceAddr := split[1]
	if split[0] == "unix" {
		unixtransport.Register(roundTripper)
		serviceAddr = "http+" + cfg.ServiceAddress + ":"
	}

	client := &http.Client{
		Transport: roundTripper,
		Timeout:   time.Second * 5,
	}

	dClient := serviceconnect.NewRaidoServiceClient(client, serviceAddr, connect.WithGRPC())

	return &Client{
		serviceClient: dClient,
	}
}

func (c *Client) TunnelAddRoute(ctx context.Context, agentId string, routes []string) error {
	_, err := c.serviceClient.TunnelAddRoute(ctx, &connect.Request[service.TunnelAddRouteRequest]{
		Msg: &service.TunnelAddRouteRequest{
			AgentId: agentId,
			Routes:  routes,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel add route: %w", err)
	}

	return nil
}

func (c *Client) TunnelRemoveRoute(ctx context.Context, agentId string, routes []string) error {
	_, err := c.serviceClient.TunnelRemoveRoute(ctx, &connect.Request[service.TunnelRemoveRouteRequest]{
		Msg: &service.TunnelRemoveRouteRequest{
			AgentId: agentId,
			Routes:  routes,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel remove route: %w", err)
	}

	return nil
}

func (c *Client) TunnelPause(ctx context.Context, agentId string) error {
	_, err := c.serviceClient.TunnelPause(ctx, &connect.Request[service.TunnelPauseRequest]{
		Msg: &service.TunnelPauseRequest{
			AgentId: agentId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel pause: %w", err)
	}

	return nil
}

func (c *Client) TunnelResume(ctx context.Context, agentId string) error {
	_, err := c.serviceClient.TunnelResume(ctx, &connect.Request[service.TunnelResumeRequest]{
		Msg: &service.TunnelResumeRequest{
			AgentId: agentId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel resume: %w", err)
	}

	return nil
}

func (c *Client) TunnelList(ctx context.Context) ([]*service.Tunnel, error) {
	resp, err := c.serviceClient.TunnelList(ctx, &connect.Request[service.Empty]{})
	if err != nil {
		return nil, fmt.Errorf("failed to request tunnels: %w", err)
	}

	return resp.Msg.GetTunnels(), nil
}

func (c *Client) AgentRemove(ctx context.Context, agentId string) error {
	_, err := c.serviceClient.AgentRemove(ctx, &connect.Request[service.AgentRemoveRequest]{
		Msg: &service.AgentRemoveRequest{
			AgentId: agentId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request agent remove: %w", err)
	}

	return nil
}

func (c *Client) ProxyStart(ctx context.Context, proxyAddr, protocol string) ([]byte, error) {
	pStartResp, err := c.serviceClient.ProxyStart(ctx, &connect.Request[service.ProxyStartRequest]{
		Msg: &service.ProxyStartRequest{
			ProxyAddress:      proxyAddr,
			TransportProtocol: protocol,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to request proxy start: %w", err)
	}

	return pStartResp.Msg.GetCertHash(), nil
}

func (c *Client) ProxyStop(ctx context.Context) error {
	_, err := c.serviceClient.ProxyStop(ctx, &connect.Request[service.Empty]{})
	if err != nil {
		return fmt.Errorf("failed to request proxy stop: %w", err)
	}

	return nil
}

func (c *Client) AgentList(ctx context.Context) (map[string]*service.Agent, error) {
	resp, err := c.serviceClient.AgentList(ctx, &connect.Request[service.Empty]{})
	if err != nil {
		return nil, fmt.Errorf("failed to request agents: %w", err)
	}

	return resp.Msg.GetAgents(), nil
}

func (c *Client) TunnelStart(ctx context.Context, agentId string, routes []string) error {
	_, err := c.serviceClient.TunnelStart(ctx, &connect.Request[service.TunnelStartRequest]{
		Msg: &service.TunnelStartRequest{
			AgentId: agentId,
			Routes:  routes,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel start: %w", err)
	}

	return nil
}

func (c *Client) TunnelStop(ctx context.Context, agentId string) error {
	_, err := c.serviceClient.TunnelStop(ctx, &connect.Request[service.TunnelStopRequest]{
		Msg: &service.TunnelStopRequest{
			AgentId: agentId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel stop: %w", err)
	}

	return nil
}
