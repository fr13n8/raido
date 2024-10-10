package main

var (
	proxyAddr   string
	serviceAddr string
	agentId     string
	proxyDomain string
	verbose     bool
)

func init() {
	serviceAddr = "unix:///var/run/raido.sock"
}
