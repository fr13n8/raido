package sysnetops

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"net"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type LinkTun struct {
	link netlink.Link
}

func NewLinkTun() (*LinkTun, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = "raido%d"

	link := &netlink.Tuntap{
		LinkAttrs: attrs,
		Mode:      netlink.TUNTAP_MODE_TUN,
	}

	if err := netlink.LinkAdd(link); err != nil {
		if os.IsExist(err) {
			log.Info().Msgf("interface \"%s\" already exists. Will reuse.", link.Attrs().Name)
			return &LinkTun{link}, nil
		}
		if os.IsNotExist(err) {
			if err := createTunDevice(); err != nil {
				return nil, fmt.Errorf("failed to create TUN device: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to add interface: %w", err)
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return nil, fmt.Errorf("failed to bring up interface \"%s\": %w", link.Attrs().Name, err)
	}

	return &LinkTun{link}, nil
}

func GetLinkTunByName(name string) (*LinkTun, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil, fmt.Errorf("interface \"%s\" does not exist", name)
		}

		return nil, fmt.Errorf("failed to get interface by name: %w", err)
	}

	return &LinkTun{link}, nil
}

func (l *LinkTun) RemoveRoute(address string) error {
	ns, err := parseNetAddress(address)
	if err != nil {
		return fmt.Errorf("error parse route: %w", err)
	}

	return l.removeRoute(ns)
}

func (l *LinkTun) AddRoute(address string) error {
	ns, err := parseNetAddress(address)
	if err != nil {
		return fmt.Errorf("error parse route: %w", err)
	}

	if err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: l.link.Attrs().Index,
		Dst:       ns.Network,
	}); err != nil && !errors.Is(err, syscall.EEXIST) && !errors.Is(err, syscall.EAFNOSUPPORT) {
		return fmt.Errorf("error netlink add route: %w", err)
	}

	return nil
}

func (l *LinkTun) SetMTU(mtu int) error {
	if err := netlink.LinkSetMTU(l.link, mtu); err != nil {
		return fmt.Errorf("error setting MTU on interface: \"%s\": %w", l.link.Attrs().Name, err)
	}

	return nil
}

func (l *LinkTun) DOWN() error {
	if err := netlink.LinkSetDown(l.link); err != nil {
		return fmt.Errorf("failed to DOWN interface \"%s\": %w", l.link.Attrs().Name, err)
	}

	return nil
}

func (l *LinkTun) Destroy() error {
	if err := netlink.LinkDel(l.link); err != nil {
		return fmt.Errorf("failed to delete interface \"%s\": %w", l.link.Attrs().Name, err)
	}

	return nil
}

func Destroy(name string) error {
	l, err := GetLinkTunByName(name)
	if err != nil {
		return fmt.Errorf("failed to get interface \"%s\": %w", name, err)
	}

	link, err := netlink.LinkByName(l.link.Attrs().Name)
	if err != nil {
		return fmt.Errorf("failed to get interface by name \"%s\": %w", l.link.Attrs().Name, err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete interface \"%s\": %w", l.link.Attrs().Name, err)
	}

	return nil
}

func (l *LinkTun) removeRoute(address NetAddress) error {
	route := &netlink.Route{
		LinkIndex: l.link.Attrs().Index,
		Dst:       address.Network,
	}

	if err := netlink.RouteDel(route); err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EAFNOSUPPORT) {
		return fmt.Errorf("error netlink remove route: %w", err)
	}

	return nil
}

type NetAddress struct {
	IP      net.IP
	Network *net.IPNet
}

func parseNetAddress(address string) (NetAddress, error) {
	ip, network, err := net.ParseCIDR(address)
	if err != nil {
		return NetAddress{}, err
	}
	return NetAddress{
		IP:      ip,
		Network: network,
	}, nil
}

func (addr NetAddress) String() string {
	maskSize, _ := addr.Network.Mask.Size()
	return fmt.Sprintf("%s/%d", addr.IP.String(), maskSize)
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
	return l.link.Attrs().Name
}

func (l *LinkTun) Routes() (map[string]netlink.Route, error) {
	routes, err := netlink.RouteList(l.link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get route list: %w", err)
	}
	tapRoutes := make(map[string]netlink.Route)
	for _, route := range routes {
		tapRoutes[route.Dst.String()] = route
	}
	return tapRoutes, nil
}

//	mkdir -p /dev/net && \
//	    mknod /dev/net/tun c 10 200 && \
//	    chmod 600 /dev/net/tun
func createTunDevice() error {
	// Create the /dev/net directory if it doesn't exist
	err := os.MkdirAll("/dev/net", 0755)
	if err != nil {
		return fmt.Errorf("failed to create /dev/net: %v", err)
	}

	// Check if the /dev/net/tun device already exists
	if _, err := os.Stat("/dev/net/tun"); os.IsNotExist(err) {
		// Create the /dev/net/tun device node with major number 10 and minor number 200
		err = unix.Mknod("/dev/net/tun", unix.S_IFCHR|0600, int(unix.Mkdev(10, 200)))
		if err != nil {
			return fmt.Errorf("failed to create /dev/net/tun: %v", err)
		}
	}

	// Change permissions of /dev/net/tun to 600
	err = os.Chmod("/dev/net/tun", 0600)
	if err != nil {
		return fmt.Errorf("failed to change permissions for /dev/net/tun: %v", err)
	}

	return nil
}
