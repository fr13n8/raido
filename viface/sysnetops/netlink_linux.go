package sysnetops

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"net"

	"github.com/fr13n8/raido/utils/ip"
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	defaultTunName   = "raido%d"
	defaultTunMode   = netlink.TUNTAP_MODE_TUN
	netDeviceDir     = "/dev/net/"
	tunDevicePath    = "/dev/net/tun"
	loopbackNetwork  = "240.0.0.0/4"
	loopbackMaskSize = 32
	netFilePermMode  = os.FileMode(0755)
)

type LinkTun struct {
	link netlink.Link
	name string
}

func NewLinkTun() (*LinkTun, error) {
	if err := ensureTunDevice(); err != nil {
		return nil, fmt.Errorf("tun device setup failed: %w", err)
	}

	attrs := netlink.NewLinkAttrs()
	attrs.Name = defaultTunName
	attrs.OperState = netlink.OperUp

	link := &netlink.Tuntap{
		LinkAttrs: attrs,
		Mode:      defaultTunMode,
	}

	err := netlink.LinkAdd(link)
	switch {
	case err == nil:
		log.Debug().Str("interface", link.Attrs().Name).Msg("Created new TUN interface")
	case os.IsExist(err):
		existing, err := netlink.LinkByName(link.Attrs().Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing TUN interface: %w", err)
		}
		link = existing.(*netlink.Tuntap)
		log.Debug().Str("interface", link.Attrs().Name).Msg("Using existing TUN interface")
	default:
		return nil, fmt.Errorf("failed to create TUN interface: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		_ = netlink.LinkDel(link)
		return nil, fmt.Errorf("failed to bring up interface \"%s\": %w", link.Attrs().Name, err)
	}

	return &LinkTun{
		link: link,
		name: link.Attrs().Name,
	}, nil
}

// add the next available network address to the interface routes from the range 240.0.0.0/4
// and each time a new interface is created the address is increased by 1 (for example, 240.1.0.0/32)
func (l *LinkTun) AddLoopbackRoute() error {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to get route list: %w", err)
	}

	var nextAddr byte
	for _, route := range routes {
		if route.Dst.IP[0] == 240 {
			if route.Dst.IP[1] > nextAddr {
				nextAddr = route.Dst.IP[1]
			}
		}
	}

	nextAddr++
	if nextAddr == 0 {
		nextAddr++
	}

	if err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: l.link.Attrs().Index,
		Dst: &net.IPNet{
			IP:   net.IPv4(240, nextAddr, 0, 0),
			Mask: net.CIDRMask(32, 32),
		},
	}); err != nil {
		return fmt.Errorf("failed to add route to interface: %w", err)
	}

	return nil
}

func GetLinkTunByName(name string) (*LinkTun, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil, fmt.Errorf("interface \"%s\" does not exist", name)
		}

		return nil, fmt.Errorf("failed to get interface by name: %w", err)
	}

	return &LinkTun{
		link: link,
		name: name,
	}, nil
}

func (l *LinkTun) RemoveRoutes(routes ...string) error {
	var errs []error
	for _, route := range routes {
		ns, err := ip.ParseNetAddress(route)
		if err != nil {
			errs = append(errs, fmt.Errorf("error parse route \"%s\": %w", route, err))
			continue
		}

		if err := l.removeRoute(ns); err != nil {
			errs = append(errs, fmt.Errorf("error netlink remove route \"%s\" from interface \"%s\": %w", route, l.name, err))
		}
	}

	return errors.Join(errs...)
}

func (l *LinkTun) AddRoutes(routes ...string) error {
	var errs []error
	for _, route := range routes {
		ns, err := ip.ParseNetAddress(route)
		if err != nil {
			errs = append(errs, fmt.Errorf("error parse route \"%s\": %w", route, err))
			continue
		}

		if err := netlink.RouteAdd(&netlink.Route{
			LinkIndex: l.link.Attrs().Index,
			Dst:       ns.Network,
		}); err != nil && !errors.Is(err, syscall.EEXIST) && !errors.Is(err, syscall.EAFNOSUPPORT) {
			errs = append(errs, fmt.Errorf("error netlink add route \"%s\" to interface \"%s\": %w", route, l.name, err))
		}
	}

	return errors.Join(errs...)
}

func (l *LinkTun) SetDown() error {
	if err := netlink.LinkSetDown(l.link); err != nil {
		return fmt.Errorf("failed to DOWN interface \"%s\": %w", l.name, err)
	}

	l.link.Attrs().OperState = netlink.OperDown
	return nil
}

func (l *LinkTun) SetUp() error {
	if err := netlink.LinkSetUp(l.link); err != nil {
		return fmt.Errorf("failed to UP interface \"%s\": %w", l.name, err)
	}

	l.link.Attrs().OperState = netlink.OperUp
	return nil
}

func (l *LinkTun) Status() string {
	return l.link.Attrs().OperState.String()
}

func (l *LinkTun) Destroy() error {
	if err := netlink.LinkDel(l.link); err != nil {
		return fmt.Errorf("failed to delete interface \"%s\": %w", l.name, err)
	}

	return nil
}

func (l *LinkTun) removeRoute(address ip.NetAddress) error {
	route := &netlink.Route{
		LinkIndex: l.link.Attrs().Index,
		Dst:       address.Network,
	}

	if err := netlink.RouteDel(route); err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EAFNOSUPPORT) {
		return fmt.Errorf("error netlink remove route: %w", err)
	}

	return nil
}

func GetTunTaps() ([]LinkTun, error) {
	tuns, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to get link list: %w", err)
	}
	var tuntaps []LinkTun
	for _, link := range tuns {
		if link.Type() == "tuntap" {
			tuntaps = append(tuntaps, LinkTun{
				link: link,
			})
		}
	}
	return tuntaps, nil
}

func (l *LinkTun) Name() string {
	return l.name
}

func (l *LinkTun) Routes() ([]string, error) {
	routes, err := netlink.RouteList(l.link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get route list: %w", err)
	}

	destinationRoutes := make([]string, 0, len(routes))
	for _, route := range routes {
		localAddress, err := ip.ParseNetAddress(route.Dst.String())
		if err != nil {
			log.Error().Err(err).Msg("could not parse local address")
			continue
		}
		if !ip.LoopbackRoute.Network.Contains(localAddress.IP) {
			destinationRoutes = append(destinationRoutes, route.Dst.String())
		}
	}
	return destinationRoutes, nil
}

func (l *LinkTun) GetLoopbackRoute() (string, error) {
	routes, err := netlink.RouteList(l.link, netlink.FAMILY_ALL)
	if err != nil {
		return "", fmt.Errorf("failed to get route list: %w", err)
	}

	for _, route := range routes {
		localAddress, err := ip.ParseNetAddress(route.Dst.String())
		if err != nil {
			log.Error().Err(err).Msg("could not parse local address")
			continue
		}
		if ip.LoopbackRoute.Network.Contains(localAddress.IP) {
			return route.Dst.String(), nil
		}
	}

	return "", nil
}

//	mkdir -p /dev/net && \
//	    mknod /dev/net/tun c 10 200 && \
//	    chmod 600 /dev/net/tun
func ensureTunDevice() error {
	// Create the /dev/net directory if it doesn't exist
	if err := os.MkdirAll(netDeviceDir, netFilePermMode); err != nil {
		return fmt.Errorf("failed to create network device directory: %w", err)
	}

	// Check if the /dev/net/tun device already exists
	if _, err := os.Stat(tunDevicePath); os.IsNotExist(err) {
		// Create the /dev/net/tun device node with major number 10 and minor number 200
		if err = unix.Mknod(tunDevicePath, unix.S_IFCHR|0600, int(unix.Mkdev(10, 200))); err != nil {
			return fmt.Errorf("failed to create tun device: %w", err)
		}
	}

	// Change permissions of /dev/net/tun to 600
	err := os.Chmod(tunDevicePath, 0600)
	if err != nil {
		return fmt.Errorf("failed to change permissions of tun device: %w", err)
	}

	return nil
}
