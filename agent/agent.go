package agent

import (
	"context"
	"fmt"

	"github.com/fr13n8/raido/proxy/tunnel"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
)

type Agent struct {
	Name      string
	Conn      quic.Connection
	Routes    []string
	Tunnel    *tunnel.Tunnel
	TunStatus bool
}

func New(name string, conn quic.Connection, routes []string) *Agent {
	return &Agent{
		Name:   name,
		Conn:   conn,
		Routes: routes,
	}
}

func (a *Agent) StartTunnel(ctx context.Context) error {
	if a.Tunnel != nil {
		return nil
	}

	tun, err := tunnel.NewTunnel(ctx, a.Conn)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	for _, r := range a.Routes {
		if err := tun.Link().AddRoute(r); err != nil {
			log.Error().Err(err).Msgf("error add route \"%s\" to interface \"%s\"", r, tun.Name())
		}
	}

	a.Tunnel = tun
	a.TunStatus = true
	return nil
}

func (a *Agent) CloseTunnel() error {
	if a.Tunnel == nil {
		return nil
	}

	a.TunStatus = false
	return a.Tunnel.Close()
}
