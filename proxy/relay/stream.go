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

func Pipe(tunnelConn, originConn io.ReadWriteCloser) error {
	return PipeBidirectional(tunnelConn, originConn)
}

func PipeBidirectional(downstream, upstream Stream) error {
	errChan := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errChan <- unidirectionalStream(downstream, upstream, "downstream->upstream")
	}()

	go func() {
		defer wg.Done()
		errChan <- unidirectionalStream(upstream, downstream, "upstream->downstream")
	}()

	wg.Wait()

	var errs []error
	for range 2 {
		if err := <-errChan; err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during bidirectional copy: %v", errors.Join(errs...))
	}

	return nil
}

func unidirectionalStream(dst WriterCloser, src Reader, dir string) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Msgf("recovered from panic in %s stream: %v", dir, err)
		}
		dst.Close()
	}()
	written, err := copyData(dst, src, dir)
	if err != nil && !IsOKNetworkError(err) {
		log.Error().Msgf("error during %s copy: %v", dir, err)
		return err
	}
	log.Debug().Msgf("copied %d bytes in %s direction", written, dir)
	return nil
}

const debugCopy = false

func copyData(dst io.Writer, src io.Reader, dir string) (written int64, err error) {
	if debugCopy {
		copyBuffer := func(dst io.Writer, src io.Reader, dir string) (written int64, err error) {
			buf := make([]byte, 32*1024)
			for {
				nr, er := src.Read(buf)
				if nr > 0 {
					fmt.Println(dir, time.Now().UnixNano(), "\n"+hex.Dump(buf[0:nr]))
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

const defaultBufferSize = 128 * 1024

var bufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, defaultBufferSize)
		return &buf
	},
}

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	if rf, ok := dst.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}

	buffer := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(buffer)

	return io.CopyBuffer(dst, src, *buffer)
}
