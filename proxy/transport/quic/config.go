package quic

import (
	"time"

	"github.com/quic-go/quic-go"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	// DefaultBackoff is the default backoff used when dialing and serving
	// a connection.
	DefaultBackoff = wait.Backoff{
		Steps:    5,
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}

	qConf = &quic.Config{
		HandshakeIdleTimeout:       5 * time.Second,
		MaxIdleTimeout:             5 * time.Second,
		KeepAlivePeriod:            1 * time.Second,
		MaxIncomingStreams:         1 << 60,
		MaxIncomingUniStreams:      -1,
		DisablePathMTUDiscovery:    false,
		MaxConnectionReceiveWindow: 30 * (1 << 20), // 30 MB
		MaxStreamReceiveWindow:     6 * (1 << 20),  // 6 MB
		// InitialPacketSize:          1252,
		Versions: []quic.Version{quic.Version2},
		// Tracer:   NewClientTracer(&log.Logger, 1),
	}
)
