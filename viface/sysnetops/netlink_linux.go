package sysnetops

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"net"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
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
		log.Error().Msgf("error link add: %v", err)
		return nil, err
	}

	if err := netlink.LinkSetUp(link); err != nil {
		log.Error().Msgf("error bringing up interface: \"%s\"", link.Attrs().Name)
		return nil, err
	}

	return &LinkTun{link}, nil
}

func GetLinkTunByName(name string) (*LinkTun, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			log.Debug().Err(os.ErrNotExist).Msgf("interface \"%s\" does not exist", name)
			return nil, os.ErrNotExist
		}

		log.Debug().Err(err).Msgf("error when check if interface \"%s\" exist", name)
		return nil, err
	}

	return &LinkTun{link}, nil
}

func (l *LinkTun) RemoveRoute(address string) error {
	ns, err := parseNetAddress(address)
	if err != nil {
		return err
	}

	return l.removeRoute(ns)
}

func (l *LinkTun) AddRoute(address string) error {
	ns, err := parseNetAddress(address)
	if err != nil {
		return err
	}

	if err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: l.link.Attrs().Index,
		Dst:       ns.Network,
	}); err != nil && !errors.Is(err, syscall.EEXIST) && !errors.Is(err, syscall.EAFNOSUPPORT) {
		log.Debug().Err(err).Msg("netlink add unreachable route:")
		return err
	}

	return nil
}

func (l *LinkTun) Close() error {
	return netlink.LinkDel(l.link)
}

func (l *LinkTun) SetMTU(mtu int) error {
	if err := netlink.LinkSetMTU(l.link, mtu); err != nil {
		log.Error().Err(err).Msgf("error setting MTU on interface: \"%s\"", l.link.Attrs().Name)
		return err
	}

	return nil
}

func (l *LinkTun) DOWN() error {
	if err := netlink.LinkSetDown(l.link); err != nil {
		log.Error().Err(err).Msgf("failed to DOWN interface \"%s\"", l.link.Attrs().Name)
		return err
	}

	return nil
}

func (l *LinkTun) Destroy() error {
	if err := netlink.LinkDel(l.link); err != nil {
		log.Debug().Err(err).Msgf("failed to delete interface \"%s\"", l.link.Attrs().Name)
		return err
	}

	return nil
}

func Destroy(name string) error {
	l, err := GetLinkTunByName(name)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get interface \"%s\"", name)
		return err
	}

	link, err := netlink.LinkByName(l.link.Attrs().Name)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get interface by name \"%s\"", l.link.Attrs().Name)
		return err
	}

	if err := netlink.LinkDel(link); err != nil {
		log.Debug().Err(err).Msgf("failed to delete interface \"%s\"", l.link.Attrs().Name)
		return err
	}

	return nil
}

func (l *LinkTun) removeRoute(address NetAddress) error {
	route := &netlink.Route{
		LinkIndex: l.link.Attrs().Index,
		Dst:       address.Network,
	}

	if err := netlink.RouteDel(route); err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EAFNOSUPPORT) {
		log.Debug().Err(err).Msg("error netlink remove route")
		return err
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
		log.Debug().Err(err).Msg("error parse route")
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
		log.Error().Err(err).Msg("failed to get link list")
		return nil, err
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

func (l *LinkTun) Routes() map[string]netlink.Route {
	routes, err := netlink.RouteList(l.link, netlink.FAMILY_ALL)
	if err != nil {
		log.Error().Err(err).Msg("failed to get route list")
		return nil
	}
	tapRoutes := make(map[string]netlink.Route)
	for _, route := range routes {
		tapRoutes[route.Dst.String()] = route
	}
	return tapRoutes
}
