package relay

import (
	"errors"
	"io"
	"net"
	"strings"
	"syscall"
)

var (
	// UseOfClosedNetworkConnection is a special string some parts of
	// go standard lib are using that is the only way to identify some errors
	UseOfClosedNetworkConnection = "use of closed network connection"
	// FailedToSendCloseNotify is an error message from Go net package
	// indicating that the connection was closed by the server.
	FailedToSendCloseNotify = "tls: failed to send closeNotify alert (but connection was closed anyway)"
)

// IsUseOfClosedNetworkError returns true if the specified error
// indicates the use of a closed network connection.
func IsUseOfClosedNetworkError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), UseOfClosedNetworkConnection)
}

// IsFailedToSendCloseNotifyError returns true if the provided error is the
// "tls: failed to send closeNotify".
func IsFailedToSendCloseNotifyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), FailedToSendCloseNotify)
}

// IsOKNetworkError returns true if the provided error received from a network
// operation is one of those that usually indicate normal connection close.
func IsOKNetworkError(err error) bool {
	// Unwrap and check if the error is wrapped or contains multiple errors.
	unwrappedErr := errors.Unwrap(err)
	if unwrappedErr != nil {
		// If the error wraps multiple errors (e.g., in a multi-error pattern),
		// recursively check if all of them are OK network errors.
		if multiErr, ok := unwrappedErr.(interface{ Errors() []error }); ok {
			for _, e := range multiErr.Errors() {
				if !IsOKNetworkError(e) {
					return false
				}
			}
			return true
		}
	}

	// Check for specific "OK" network errors.
	return errors.Is(err, io.EOF) || IsUseOfClosedNetworkError(err) || IsFailedToSendCloseNotifyError(err)
}

func IsHostResponded(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errors.Is(errno, syscall.ECONNREFUSED) || errors.Is(errno, syscall.ECONNRESET) || errors.Is(errno, syscall.ECONNABORTED)
	}
	return false
}
