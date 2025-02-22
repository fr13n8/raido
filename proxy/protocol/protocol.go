package protocol

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

// Constants representing the VPN name and transport protocols (TCP, UDP, ICMP).
const (
	Name = "raido"
)

const (
	TransportTCP = uint8(iota)
	TransportUDP
	TransportICMP
)

const (
	Networkv4 = uint8(iota)
	Networkv6
)

// IPAddressWithPortProtocol holds IP, port, protocol, and network information for both IPv4 and IPv6.
type IPAddressWithPortProtocol struct {
	IP       net.IP // Can hold both IPv4 (4 bytes) or IPv6 (16 bytes)
	Port     uint16 // 16 bits for the port
	Protocol uint8  // 2 bits for protocol (TCP, UDP, ICMP)
	Network  uint8  // 2 bits for network version (v4 or v6)
}

// Encode encodes the structure into a byte slice that contains all the necessary information.
func (ipStruct *IPAddressWithPortProtocol) Encode() ([]byte, error) {
	// Encode the protocol and network in the first byte
	header := (ipStruct.Network&0x03)<<6 | (ipStruct.Protocol&0x03)<<4
	result := []byte{header}

	// Append the port (16 bits)
	result = append(result, byte(ipStruct.Port>>8), byte(ipStruct.Port&0xFF))

	// Append the IP address (either 4 bytes for IPv4 or 16 bytes for IPv6)
	switch ipStruct.Network {
	case Networkv4:
		if ip4 := ipStruct.IP.To4(); ip4 != nil {
			result = append(result, ip4...)
		} else {
			return nil, fmt.Errorf("invalid IPv4 address")
		}
	case Networkv6:
		if ip6 := ipStruct.IP.To16(); ip6 != nil {
			result = append(result, ip6...)
		} else {
			return nil, fmt.Errorf("invalid IPv6 address")
		}
	default:
		return nil, fmt.Errorf("unknown network type")
	}

	return result, nil
}

// Decode decodes the byte slice back into the IPAddressWithPortProtocol structure.
func Decode(encoded []byte) (*IPAddressWithPortProtocol, error) {
	if len(encoded) < 3 {
		return nil, fmt.Errorf("encoded data too short")
	}

	// Extract the header byte
	header := encoded[0]
	network := (header >> 6) & 0x03
	protocol := (header >> 4) & 0x03

	// Extract the port (16 bits)
	port := uint16(encoded[1])<<8 | uint16(encoded[2])

	// Determine the IP address length based on the network type
	var ip net.IP
	switch network {
	case Networkv4:
		if len(encoded) < 7 {
			return nil, fmt.Errorf("encoded data too short for IPv4")
		}
		ip = net.IPv4(encoded[3], encoded[4], encoded[5], encoded[6])
	case Networkv6:
		if len(encoded) < 19 {
			return nil, fmt.Errorf("encoded data too short for IPv6")
		}
		ip = net.IP(encoded[3:19])
	default:
		return nil, fmt.Errorf("unknown network type")
	}

	return &IPAddressWithPortProtocol{
		IP:       ip,
		Port:     port,
		Protocol: protocol,
		Network:  network,
	}, nil
}

// Constants representing QUIC application error codes.
const (
	ApplicationOK = 0x0
)

// Commands for communication within the VPN protocol.
const (
	GetRoutesReqCmd        = "GetRoutesReq"
	EstablishConnectionCmd = "EstablishConnection"
)

// GetRoutesResp holds the name and list of routes for the VPN.
type GetRoutesResp struct {
	Name   string
	Routes []string
}

// Data represents the data structure sent over the protocol.
type Data struct {
	Command string
	Body    []byte
}

// ConnectResponse indicates whether a connection was successfully established.
type ConnectResponse struct {
	Established bool
}

// Decoder wraps the gob decoder for a specific type.
type Decoder[T any] struct {
	dec *gob.Decoder
	r   io.Closer
}

// NewDecoder initializes a new Decoder for type T.
func NewDecoder[T any](rd io.ReadCloser) Decoder[T] {
	return Decoder[T]{dec: gob.NewDecoder(rd), r: rd}
}

// Decode decodes the next value from the stream.
func (d Decoder[T]) Decode() (T, error) {
	var t T
	err := d.dec.Decode(&t)
	return t, err
}

// Close closes the underlying reader.
func (d Decoder[T]) Close() error {
	return d.r.Close()
}

// Encoder wraps the gob encoder for a specific type.
type Encoder[T any] struct {
	enc *gob.Encoder
	w   io.Closer
}

// NewEncoder initializes a new Encoder for type T.
func NewEncoder[T any](wr io.WriteCloser) Encoder[T] {
	return Encoder[T]{enc: gob.NewEncoder(wr), w: wr}
}

// Encode encodes the provided value to the stream.
func (e Encoder[T]) Encode(t T) error {
	return e.enc.Encode(&t)
}

// Close closes the underlying writer.
func (e Encoder[T]) Close() error {
	return e.w.Close()
}
