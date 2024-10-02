package config

import "time"

type ProxyServer struct {
	Address   string
	TLSConfig *TLSConfig
}

type ServiceServer struct {
	Address   string
	TLSConfig *TLSConfig
}

type Dialer struct {
	ProxyAddress string
	TLSConfig    *TLSConfig
}

type TLSConfig struct {
	CertFile           string
	KeyFile            string
	CAFile             string
	InsecureSkipVerify bool
	ServerName         string

	// for auto-generated default certificate.
	Validity     time.Duration
	CommonName   string
	Organization string
}
