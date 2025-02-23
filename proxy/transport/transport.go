package transport

import (
	"context"
	"net"
)

// Stream represents a bidirectional stream (e.g., QUIC stream, TCP connection, or multiplexed stream).
type Stream interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
}

// StreamConn represents a connection that can open or accept multiple streams.
type StreamConn interface {
	OpenStream(ctx context.Context) (Stream, error)
	AcceptStream(ctx context.Context) (Stream, error)
	Close() error
	CloseWithError(code uint64, reason string) error

	// GetStream and PutStream are used to manage streams in a pool.
	GetStream(ctx context.Context) (Stream, error)
	PutStream(stream Stream)
}

// StreamListener represents a listener that accepts StreamConn instances.
type StreamListener interface {
	Accept(ctx context.Context) (StreamConn, error)
	Close() error
	Addr() net.Addr
}

// Transport defines the interface for establishing connections and managing streams.
type Transport interface {
	Dial(ctx context.Context, addr string) (StreamConn, error)
	Listen(ctx context.Context, addr string) (StreamListener, error)
}
