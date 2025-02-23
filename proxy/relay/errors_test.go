package relay

import (
	"errors"
	"io"
	"net"
	"syscall"
	"testing"
)

func TestIsUseOfClosedNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "network closed error",
			err:  net.ErrClosed,
			want: true,
		},
		{
			name: "use of closed network connection error string",
			err:  errors.New("use of closed network connection"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUseOfClosedNetworkError(tt.err); got != tt.want {
				t.Errorf("IsUseOfClosedNetworkError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFailedToSendCloseNotifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "close notify error",
			err:  errors.New("tls: failed to send closeNotify alert (but connection was closed anyway)"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFailedToSendCloseNotifyError(tt.err); got != tt.want {
				t.Errorf("IsFailedToSendCloseNotifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsOKNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "EOF error",
			err:  io.EOF,
			want: true,
		},
		{
			name: "network closed error",
			err:  net.ErrClosed,
			want: true,
		},
		{
			name: "close notify error",
			err:  errors.New("tls: failed to send closeNotify alert (but connection was closed anyway)"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsOKNetworkError(tt.err); got != tt.want {
				t.Errorf("IsOKNetworkError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHostResponded(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "connection refused",
			err:  syscall.ECONNREFUSED,
			want: true,
		},
		{
			name: "connection reset",
			err:  syscall.ECONNRESET,
			want: true,
		},
		{
			name: "connection aborted",
			err:  syscall.ECONNABORTED,
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHostResponded(tt.err); got != tt.want {
				t.Errorf("IsHostResponded() = %v, want %v", got, tt.want)
			}
		})
	}
}
