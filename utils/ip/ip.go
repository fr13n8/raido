package ip

import (
	"fmt"
	"net"
)

var (
	LoopbackRoute   NetAddress
	loopbackNetwork = "240.0.0.0/4"
)

func init() {
	LoopbackRoute, _ = ParseNetAddress(loopbackNetwork)
}

type NetAddress struct {
	IP      net.IP
	Network *net.IPNet
}

func ParseNetAddress(address string) (NetAddress, error) {
	ip, network, err := net.ParseCIDR(address)
	if err != nil {
		return NetAddress{}, fmt.Errorf("failed to parse address: %w", err)
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
