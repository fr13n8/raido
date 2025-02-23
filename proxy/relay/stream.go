package relay

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Stream interface {
	Reader
	WriterCloser
}

type Reader interface {
	io.ReadCloser
}

type WriterCloser interface {
	io.WriteCloser
}

type bidirectionalStreamStatus struct {
	doneChan chan struct{}
}

func newBiStreamStatus() *bidirectionalStreamStatus {
	return &bidirectionalStreamStatus{
		doneChan: make(chan struct{}, 2),
	}
}

func (s *bidirectionalStreamStatus) markUniStreamDone() {
	s.doneChan <- struct{}{}
}

func Pipe(tunnelConn, originConn io.ReadWriteCloser) {
	PipeBidirectional(tunnelConn, originConn)
}

func PipeBidirectional(downstream, upstream Stream) error {
	status := newBiStreamStatus()

	go unidirectionalStream(downstream, upstream, "upstream->downstream", status)
	go unidirectionalStream(upstream, downstream, "downstream->upstream", status)

	for i := 0; i < 2; i++ {
		<-status.doneChan
	}

	return nil
}

func unidirectionalStream(dst WriterCloser, src Reader, dir string, status *bidirectionalStreamStatus) {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Msgf("recovered from panic in %s stream: %v", dir, err)
		}
	}()
	defer dst.Close()

	_, err := copyData(dst, src, dir)
	if err != nil && !IsOKNetworkError(err) {
		log.Error().Msgf("error during %s copy: %v", dir, err)
	}
	status.markUniStreamDone()
}

const debugCopy = false

func copyData(dst io.Writer, src io.Reader, dir string) (written int64, err error) {
	if debugCopy {
		copyBuffer := func(dst io.Writer, src io.Reader, dir string) (written int64, err error) {
			var buf []byte
			size := 32 * 1024
			buf = make([]byte, size)
			for {
				t := time.Now()
				nr, er := src.Read(buf)
				if nr > 0 {
					fmt.Println(dir, t.UnixNano(), "\n"+hex.Dump(buf[0:nr]))
					nw, ew := dst.Write(buf[0:nr])
					if nw < 0 || nr < nw {
						nw = 0
						if ew == nil {
							ew = errors.New("invalid write")
						}
					}
					written += int64(nw)
					if ew != nil {
						err = ew
						break
					}
					if nr != nw {
						err = io.ErrShortWrite
						break
					}
				}
				if er != nil {
					if er != io.EOF {
						err = er
					}
					break
				}
			}
			return written, err
		}
		return copyBuffer(dst, src, dir)
	}
	return Copy(dst, src)
}

const defaultBufferSize = 64 * 1024

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, defaultBufferSize)
	},
}

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	if rf, ok := dst.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}

	buffer := bufferPool.Get().([]byte)
	defer bufferPool.Put(&buffer)
	return io.CopyBuffer(dst, src, buffer)
}
