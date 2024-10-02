//go:build linux

package device

import (
	"context"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/rawfile"
	"gvisor.dev/gvisor/pkg/tcpip/link/fdbased"
	"gvisor.dev/gvisor/pkg/tcpip/link/tun"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type NetTun struct {
	name string
	fd   int
	mtu  uint32
	lep  stack.LinkEndpoint
}

// Open initializes the TUN device, retrieves the MTU, and creates the LinkEndpoint.
func Open(name string) (TUNDevice, error) {
	if len(name) >= unix.IFNAMSIZ {
		log.Error().Msgf("interface name too long: \"%s\"", name)
		return nil, fmt.Errorf("interface name too long: \"%s\"", name)
	}

	// Retrieve the MTU of the network interface.
	_mtu, err := rawfile.GetMTU(name)
	if err != nil {
		log.Error().Err(err).Msgf("get mtu of \"%s\" error", name)
		return nil, err
	}

	// Open the TUN device file descriptor.
	fd, err := tun.Open(name)
	if err != nil {
		log.Error().Err(err).Msgf("open TUN interface error \"%s\"", name)
		return nil, err
	}

	// Create a new LinkEndpoint using the fdbased package, setting options for performance.
	lep, err := fdbased.New(&fdbased.Options{
		FDs:            []int{fd}, // File descriptor for the TUN interface.
		MTU:            1280,      // MTU of the device.
		EthernetHeader: false,     // TUN devices don't use Ethernet headers.
		// PacketDispatchMode: fdbased.RecvMMsg, // Use MMsg for high throughput packet processing.
		// GSOMaxSize:         65536,            // Enable GSO to batch packets for higher throughput.
	})
	if err != nil {
		log.Error().Err(err).Msg("create endpoint error")
		// Ensure the fd is closed on error.
		unix.Close(fd)
		return nil, err
	}

	return &NetTun{
		name: name,
		mtu:  _mtu,
		fd:   fd,
		lep:  lep,
	}, nil
}

// Close gracefully closes the TUN device and its associated resources.
func (t *NetTun) Close() error {
	defer t.lep.Close()
	// Close the file descriptor for the TUN device.
	return unix.Close(t.fd)
}

// Dev returns the LinkEndpoint for the TUN device.
func (t *NetTun) Dev() stack.LinkEndpoint {
	return t.lep
}

// Name returns the name of the TUN device.
func (t *NetTun) Name() string {
	return t.name
}

// AddSubnet adds a subnet route to the TUN device.
func (t *NetTun) AddSubnet(context.Context, *net.IPNet) error {
	return nil
}

// RemoveSubnet removes a subnet route from the TUN device.
func (t *NetTun) RemoveSubnet(context.Context, *net.IPNet) error {
	return nil
}
