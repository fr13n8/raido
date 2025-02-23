package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/fr13n8/raido/proxy/transport"
	"github.com/hashicorp/yamux"
)

// TCPTransport implements the Transport interface for TCP with yamux multiplexing.
type TCPTransport struct {
	tlsConfig *tls.Config // Optional TLS configuration
}

// NewTCPTransport creates a new TCPTransport instance.
func NewTCPTransport(tlsConfig *tls.Config) *TCPTransport {
	return &TCPTransport{tlsConfig: tlsConfig}
}

// Dial establishes a TCP connection and wraps it with a yamux session.
func (t *TCPTransport) Dial(ctx context.Context, addr string) (transport.StreamConn, error) {
	var conn net.Conn
	var err error
	if t.tlsConfig != nil {
		conn, err = tls.Dial("tcp", addr, t.tlsConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("could not dial address: %w", err)
	}
	session, err := yamux.Client(conn, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("could not establish yamux session: %w", err)
	}

	streamConn := &TCPStreamConn{session: session}
	streamConn.streamPool = transport.NewStreamPool(16, streamConn)
	return streamConn, nil
}

// Listen sets up a TCP listener and wraps accepted connections with yamux.
func (t *TCPTransport) Listen(ctx context.Context, addr string) (transport.StreamListener, error) {
	var listener net.Listener
	var err error
	if t.tlsConfig != nil {
		listener, err = tls.Listen("tcp", addr, t.tlsConfig)
	} else {
		listener, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	return &TCPStreamListener{listener: listener}, nil
}

// TCPStreamConn wraps a yamux session as a StreamConn.
type TCPStreamConn struct {
	session    *yamux.Session
	streamPool *transport.StreamPool
}

func (c *TCPStreamConn) OpenStream(ctx context.Context) (transport.Stream, error) {
	return c.session.OpenStream()
}

func (c *TCPStreamConn) AcceptStream(ctx context.Context) (transport.Stream, error) {
	return c.session.AcceptStream()
}

func (c *TCPStreamConn) Close() error {
	return c.session.Close()
}

func (c *TCPStreamConn) CloseWithError(code uint64, reason string) error {
	return c.session.Close()
}

func (c *TCPStreamConn) GetStream(ctx context.Context) (transport.Stream, error) {
	return c.streamPool.Get(ctx)
}

func (c *TCPStreamConn) PutStream(stream transport.Stream) {
	c.streamPool.Put(stream)
}

// TCPStreamListener wraps a net.Listener to produce yamux sessions.
type TCPStreamListener struct {
	listener net.Listener
}

func (l *TCPStreamListener) Accept(ctx context.Context) (transport.StreamConn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	session, err := yamux.Server(conn, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}

	streamConn := &TCPStreamConn{session: session}
	streamConn.streamPool = transport.NewStreamPool(16, streamConn)
	return streamConn, nil
}

func (l *TCPStreamListener) Close() error {
	return l.listener.Close()
}

func (l *TCPStreamListener) Addr() net.Addr {
	return l.listener.Addr()
}
