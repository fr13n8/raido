package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	pb "github.com/fr13n8/raido/proto/service"
	"github.com/fr13n8/raido/proto/service/serviceconnect"

	"connectrpc.com/connect"
	"github.com/fr13n8/raido/config"
	"github.com/peterbourgon/unixtransport"
	"github.com/quic-go/quic-go/http3"
)

type ClientKey struct{}

type Client struct {
	serviceClient serviceconnect.RaidoServiceClient
	qConn         *http3.RoundTripper
}

func NewClient(ctx context.Context, cfg *config.ServiceDialer) *Client {
	roundTripper := &http.Transport{
		ForceAttemptHTTP2: true,
	}

	unixtransport.Register(roundTripper)

	client := &http.Client{
		Transport: roundTripper,
		Timeout:   time.Second * 5,
	}

	dClient := serviceconnect.NewRaidoServiceClient(client, "http+"+cfg.ServiceAddress+":", connect.WithGRPC())

	return &Client{
		serviceClient: dClient,
	}
}

func (c *Client) ProxyStart(ctx context.Context, proxyAddr string) ([]byte, error) {
	pStartResp, err := c.serviceClient.ProxyStart(ctx, &connect.Request[pb.ProxyStartRequest]{
		Msg: &pb.ProxyStartRequest{
			ProxyAddress: proxyAddr,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to request proxy start: %w", err)
	}

	return pStartResp.Msg.GetCertHash(), nil
}

func (c *Client) ProxyStop(ctx context.Context) error {
	_, err := c.serviceClient.ProxyStop(ctx, &connect.Request[pb.ProxyStopRequest]{})
	if err != nil {
		return fmt.Errorf("failed to request proxy stop: %w", err)
	}

	return nil
}

func (c *Client) GetAgents(ctx context.Context) (map[string]*pb.Agent, error) {
	resp, err := c.serviceClient.GetAgents(ctx, &connect.Request[pb.GetAgentsRequest]{})
	if err != nil {
		return nil, fmt.Errorf("failed to request agents: %w", err)
	}

	return resp.Msg.GetAgents(), nil
}

func (c *Client) AgentTunnelStart(ctx context.Context, agentId string) error {
	_, err := c.serviceClient.AgentTunnelStart(ctx, &connect.Request[pb.AgentTunnelStartRequest]{
		Msg: &pb.AgentTunnelStartRequest{
			AgentId: agentId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel start: %w", err)
	}

	return nil
}

func (c *Client) AgentTunnelStop(ctx context.Context, agentId string) error {
	_, err := c.serviceClient.AgentTunnelStop(ctx, &connect.Request[pb.AgentTunnelStopRequest]{
		Msg: &pb.AgentTunnelStopRequest{
			AgentId: agentId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to request tunnel stop: %w", err)
	}

	return nil
}
