package quic

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/fr13n8/raido/proxy/transport"
	"github.com/quic-go/quic-go"
)

// QUICTransport implements the Transport interface for QUIC.
type QUICTransport struct {
	tlsConfig  *tls.Config
	quicConfig *quic.Config
}

// NewQUICTransport creates a new QUICTransport instance.
func NewQUICTransport(tlsConfig *tls.Config) *QUICTransport {
	return &QUICTransport{tlsConfig: tlsConfig, quicConfig: qConfig}
}

func (t *QUICTransport) Dial(ctx context.Context, addr string) (transport.StreamConn, error) {
	conn, err := quic.DialAddr(ctx, addr, t.tlsConfig, t.quicConfig)
	if err != nil {
		return nil, err
	}
	return &QUICStreamConn{conn: conn}, nil
}

func (t *QUICTransport) Listen(ctx context.Context, addr string) (transport.StreamListener, error) {
	listener, err := quic.ListenAddr(addr, t.tlsConfig, t.quicConfig)
	if err != nil {
		return nil, err
	}
	return &QUICStreamListener{listener: listener}, nil
}

// QUICStreamConn wraps a quic.Connection as a StreamConn.
type QUICStreamConn struct {
	conn quic.Connection
}

func (c *QUICStreamConn) OpenStream(ctx context.Context) (transport.Stream, error) {
	return c.conn.OpenStreamSync(ctx)
}

func (c *QUICStreamConn) AcceptStream(ctx context.Context) (transport.Stream, error) {
	return c.conn.AcceptStream(ctx)
}

func (c *QUICStreamConn) Close() error {
	return c.conn.CloseWithError(0, "")
}

func (c *QUICStreamConn) CloseWithError(code uint64, reason string) error {
	return c.conn.CloseWithError(quic.ApplicationErrorCode(code), reason)
}

// QUICStreamListener wraps a quic.Listener as a StreamListener.
type QUICStreamListener struct {
	listener *quic.Listener
}

func (l *QUICStreamListener) Accept(ctx context.Context) (transport.StreamConn, error) {
	conn, err := l.listener.Accept(ctx)
	if err != nil {
		return nil, err
	}
	return &QUICStreamConn{conn: conn}, nil
}

func (l *QUICStreamListener) Close() error {
	return l.listener.Close()
}

func (l *QUICStreamListener) Addr() net.Addr {
	return l.listener.Addr()
}
