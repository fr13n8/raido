package proxy

import (
	"time"

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
)
