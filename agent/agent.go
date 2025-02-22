package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/transport"
	"github.com/fr13n8/raido/proxy/tunnel"
	"github.com/lithammer/shortuuid/v4"
)

type Agent struct {
	ID       string
	Hostname string
	conn     transport.StreamConn
	mu       sync.RWMutex
	routes   []string
	tunnel   *tunnel.Tunnel
}

func New(name string, conn transport.StreamConn, routes []string) *Agent {
	agentiId := shortuuid.New()
	return &Agent{
		ID:       agentiId,
		Hostname: name,
		conn:     conn,
		routes:   routes,
	}
}

func (a *Agent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.tunnel != nil {
		if err := a.tunnel.Close(); err != nil {
			return fmt.Errorf("failed to close tunnel: %w", err)
		}
	}

	if err := a.conn.CloseWithError(protocol.ApplicationOK, "server closing down"); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

func (a *Agent) TunnelStart(ctx context.Context, routes []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.tunnel != nil {
		return nil
	}

	tun, err := tunnel.NewTunnel(ctx, a.conn)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}
	a.tunnel = tun

	if len(routes) == 0 {
		routes = a.routes
	}

	if err := tun.AddRoutes(routes...); err != nil {
		return fmt.Errorf("failed to add routes to tunnel: %w", err)
	}

	return nil
}

func (a *Agent) TunnelClose() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.tunnel == nil {
		return nil
	}

	t := a.tunnel
	a.tunnel = nil

	return t.Close()
}

func (a *Agent) TunnelAddRoutes(routes ...string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	return a.tunnel.AddRoutes(routes...)
}

func (a *Agent) TunnelRemoveRoutes(routes ...string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	return a.tunnel.RemoveRoutes(routes...)
}

func (a *Agent) TunnelPause() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	return a.tunnel.Pause()
}

func (a *Agent) TunnelResume() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	return a.tunnel.Resume()
}

func (a *Agent) TunnelActiveRoutes() ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return nil, fmt.Errorf("tunnel is not initialized")
	}

	return a.tunnel.ActiveRoutes()
}

func (a *Agent) TunnelLoopbackRoute() (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return "", fmt.Errorf("tunnel is not initialized")
	}

	return a.tunnel.GetLoopbackRoute()
}

func (a *Agent) TunnelStatus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return "disconnected"
	}

	return a.tunnel.Status()
}

func (a *Agent) TunnelName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.tunnel == nil {
		return ""
	}

	return a.tunnel.Name()
}

func (a *Agent) Routes() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.routes
}
