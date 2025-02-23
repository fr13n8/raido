package relay

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type mockStream struct {
	readData  []byte
	writeData []byte
	closed    bool
	readErr   error
	writeErr  error
	mu        sync.Mutex
}

func newMockStream(data []byte) *mockStream {
	return &mockStream{
		readData:  data,
		writeData: make([]byte, 0, len(data)),
	}
}

func (m *mockStream) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, errors.New("use of closed network connection")
	}
	if m.readErr != nil {
		return 0, m.readErr
	}
	if len(m.readData) == 0 {
		return 0, io.EOF
	}
	n = copy(p, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

func (m *mockStream) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, errors.New("use of closed network connection")
	}
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.writeData = append(m.writeData, p...)
	return len(p), nil
}

func (m *mockStream) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

func TestPipeBidirectional(t *testing.T) {
	tests := []struct {
		name           string
		downstreamData []byte
		upstreamData   []byte
		expectError    bool
	}{
		{
			name:           "Basic bidirectional copy",
			downstreamData: []byte("downstream data"),
			upstreamData:   []byte("upstream data"),
			expectError:    false,
		},
		{
			name:           "Empty data transfer",
			downstreamData: []byte{},
			upstreamData:   []byte{},
			expectError:    false,
		},
		{
			name:           "Large data transfer",
			downstreamData: bytes.Repeat([]byte("D"), 16000), // 16KB downstream
			upstreamData:   bytes.Repeat([]byte("U"), 16000), // 16KB upstream
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downstream := newMockStream(tt.downstreamData)
			upstream := newMockStream(tt.upstreamData)

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				err := PipeBidirectional(downstream, upstream)
				if tt.expectError && err == nil {
					t.Error("expected error but got nil")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}()

			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Success
			case <-time.After(5 * time.Second):
				t.Fatal("test timed out waiting for PipeBidirectional to complete")
			}

			if !bytes.Equal(downstream.writeData, tt.upstreamData) {
				t.Errorf("downstream received incorrect data.\nwant: %v\ngot: %v", tt.upstreamData, downstream.writeData)
			}
			if !bytes.Equal(upstream.writeData, tt.downstreamData) {
				t.Errorf("upstream received incorrect data.\nwant: %v\ngot: %v", tt.downstreamData, upstream.writeData)
			}
		})
	}
}

func TestCopy(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "Basic copy",
			data:        []byte("test data"),
			expectError: false,
		},
		{
			name:        "Empty data",
			data:        []byte{},
			expectError: false,
		},
		{
			name:        "Large data",
			data:        bytes.Repeat([]byte("large data block "), 1000),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := bytes.NewReader(tt.data)
			dst := &bytes.Buffer{}

			written, err := Copy(dst, src)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if written != int64(len(tt.data)) {
				t.Errorf("incorrect number of bytes written. want: %d, got: %d", len(tt.data), written)
			}
			if !bytes.Equal(dst.Bytes(), tt.data) {
				t.Errorf("copied data doesn't match.\nwant: %v\ngot: %v", tt.data, dst.Bytes())
			}
		})
	}
}

func TestUnidirectionalStream(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		injectError error
		expectError bool
	}{
		{
			name:        "Normal operation",
			data:        []byte("test data"),
			expectError: false,
		},
		{
			name:        "Handle network error",
			data:        []byte("test data"),
			injectError: errors.New("use of closed network connection"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := newMockStream(tt.data)
			dst := newMockStream(nil)

			if tt.injectError != nil {
				src.readErr = tt.injectError
			}

			done := make(chan struct{})
			go func() {
				unidirectionalStream(dst, src, "test")
				close(done)
			}()

			select {
			case <-done:
				// Success
			case <-time.After(time.Second):
				t.Error("unidirectionalStream did not complete in time")
			}

			if !tt.expectError && !bytes.Equal(dst.writeData, tt.data) {
				t.Errorf("data was not copied correctly.\nwant: %v\ngot: %v", tt.data, dst.writeData)
			}
		})
	}
}
