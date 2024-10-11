package agent

import (
	"context"
	"fmt"

	"github.com/fr13n8/raido/proxy/tunnel"
	"github.com/quic-go/quic-go"
)

type Agent struct {
	Name   string
	Conn   quic.Connection
	Routes []string
	Tunnel *tunnel.Tunnel
}

func New(name string, conn quic.Connection, routes []string) *Agent {
	return &Agent{
		Name:   name,
		Conn:   conn,
		Routes: routes,
	}
}

func (a *Agent) StartTunnel(ctx context.Context, routes []string) error {
	if a.Tunnel != nil {
		return nil
	}

	tun, err := tunnel.NewTunnel(ctx, a.Conn)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	if len(routes) == 0 {
		routes = a.Routes
	}

	a.Tunnel = tun

	if err := tun.AddRoutes(routes...); err != nil {
		return fmt.Errorf("failed to add routes to tunnel: %w", err)
	}

	return nil
}

func (a *Agent) CloseTunnel() error {
	if a.Tunnel == nil {
		return nil
	}

	t := a.Tunnel
	a.Tunnel = nil

	return t.Close()
}
