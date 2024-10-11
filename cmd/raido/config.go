package main

var (
	proxyAddr   string
	serviceAddr string
	agentId     string
	routes      []string
	proxyDomain string
)

func init() {
	serviceAddr = "unix:///var/run/raido.sock"
}
