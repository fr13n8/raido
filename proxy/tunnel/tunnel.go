package tunnel

import (
	"context"

	"github.com/fr13n8/raido/viface/device"
	"github.com/fr13n8/raido/viface/netstack"
	"github.com/fr13n8/raido/viface/sysnetops"
	"github.com/quic-go/quic-go"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type Tunnel struct {
	stack  *stack.Stack
	device device.TUNDevice
	link   *sysnetops.LinkTun
}

func NewTunnel(ctx context.Context, conn quic.Connection) (*Tunnel, error) {
	link, err := sysnetops.NewLinkTun()
	if err != nil {
		return nil, err
	}

	tun, err := device.Open(link.Name())
	if err != nil {
		return nil, err
	}

	s, err := netstack.NewNetStack(ctx, tun.Dev(), conn)
	if err != nil {
		return nil, err
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
		return err
	}

	return nil
}

func (t *Tunnel) Name() string {
	return t.link.Name()
}

func (t *Tunnel) Link() *sysnetops.LinkTun {
	return t.link
}
