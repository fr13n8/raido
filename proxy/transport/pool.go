package transport

import (
	"context"
)

type StreamPool struct {
	streams chan Stream
	conn    StreamConn
}

func NewStreamPool(size int, conn StreamConn) *StreamPool {
	pool := &StreamPool{
		streams: make(chan Stream, size),
		conn:    conn,
	}
	// Prepopulate the pool with streams
	for range size {
		stream, err := conn.OpenStream(context.Background())
		if err == nil {
			pool.streams <- stream
		}
	}
	return pool
}

func (p *StreamPool) Get(ctx context.Context) (Stream, error) {
	select {
	case stream := <-p.streams:
		return stream, nil
	default:
		// If the pool is empty, create a new stream
		return p.conn.OpenStream(ctx)
	}
}

func (p *StreamPool) Put(stream Stream) {
	select {
	case p.streams <- stream:
		// Stream returned to the pool
	default:
		// Pool is full, close the stream
		stream.Close()
	}
}
