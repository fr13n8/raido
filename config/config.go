package config

import (
	"crypto/tls"
	"time"
)

var (
	ShutdownTimeout = 2 * time.Second
	RaidoPath       = "/etc/raido"
)

type ProxyServer struct {
	Address   string
	TLSConfig *tls.Config
}

type ProxyDialer struct {
	ProxyAddress string
	RetryCount   uint8
	TLSConfig    *tls.Config
}

type ServiceServer struct {
	Address   string
	TLSConfig *tls.Config
}

type ServiceDialer struct {
	ServiceAddress string
	TLSConfig      *tls.Config
}
