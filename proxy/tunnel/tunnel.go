package tunnel

import (
	"context"
	"fmt"

	"github.com/fr13n8/raido/viface/device"
	"github.com/fr13n8/raido/viface/netstack"
	"github.com/fr13n8/raido/viface/sysnetops"
	"github.com/quic-go/quic-go"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type Tunnel struct {
	stack        *stack.Stack
	device       device.TUNDevice
	link         *sysnetops.LinkTun
	activeRoutes []string
}

func NewTunnel(ctx context.Context, conn quic.Connection) (*Tunnel, error) {
	link, err := sysnetops.NewLinkTun()
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %w", err)
	}

	tun, err := device.Open(link.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to open TUN device: %w", err)
	}

	s, err := netstack.NewNetStack(ctx, tun.Dev(), conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create network stack: %w", err)
	}

	return &Tunnel{
		stack:  s,
		link:   link,
		device: tun,
	}, nil
}

func (t *Tunnel) Close() error {
	t.device.Dev().Close()
	t.stack.Close()

	if err := t.link.Destroy(); err != nil {
		return fmt.Errorf("failed to destroy TUN device: %w", err)
	}

	return nil
}

func (t *Tunnel) Name() string {
	return t.link.Name()
}

func (t *Tunnel) AddRoutes(routes ...string) error {
	if t.link == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	if err := t.link.AddRoutes(routes...); err != nil {
		return fmt.Errorf("failed to add routes: %w", err)
	}

	t.activeRoutes = append(t.activeRoutes, routes...)

	return nil
}

func (t *Tunnel) RemoveRoutes(routes ...string) error {
	if t.link == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	if err := t.link.RemoveRoutes(routes...); err != nil {
		return fmt.Errorf("failed to remove routes: %w", err)
	}

	var newRoutes []string
	for _, r := range t.activeRoutes {
		for _, route := range routes {
			if r == route {
				continue
			}

			newRoutes = append(newRoutes, r)
		}
	}

	t.activeRoutes = newRoutes

	return nil
}

func (t *Tunnel) Pause() error {
	if t.link == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	return t.link.SetDown()
}

func (t *Tunnel) Resume() error {
	if t.link == nil {
		return fmt.Errorf("tunnel is not initialized")
	}

	if err := t.link.SetUp(); err != nil {
		return fmt.Errorf("failed to bring up interface: %w", err)
	}

	if err := t.AddRoutes(t.activeRoutes...); err != nil {
		return fmt.Errorf("failed to add active routes: %w", err)
	}

	return nil
}

func (t *Tunnel) ActiveRoutes() ([]string, error) {
	routes, err := t.link.Routes()
	if err != nil {
		return nil, fmt.Errorf("failed to get active routes: %w", err)
	}

	return routes, nil
}

func (t *Tunnel) Status() string {
	return t.link.Status()
}
