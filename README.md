<h1 align="center">
  <img width="250" src="./doc/raido.png" alt="raido logo" />
</h1>

<h3 align="center">Raido is a “VPN-like” reverse proxy server with tunneling traffic through QUIC to access private network</h3>

<p align="center">
  <a href="https://fr13n8.github.io/blog/"><img src="https://img.shields.io/badge/made%20by-fr13n8-blue.svg?style=flat-square" /></a>
  <img src="https://img.shields.io/badge/go%20version-%3E=1.23-61CFDD.svg?style=flat-square" />
  <img src="https://goreportcard.com/badge/github.com/fr13n8/raido" />
</p>

<div align="center">
  <img width="100%" src="./doc/diagram.svg" />
</div>

---

> [!WARNING]
> **The functionality was tested only on Linux machines.**

## Features

- Application
  - No Wireguard, SOCKS, Proxychains
  - Userspace network stack with gVisor
  - Traffic tunneling over QUIC
  - Easy to use
  - Possible to run in daemon mode
  - Automatic management of **TUN** interfaces
  - Self-signed certificates
- Network
  - TCP
  - UDP

## Requirements

### Agent side

Bidirectional UDP access to proxy on one port.

### Proxy side

Privileged access to create and configure the **TUN** interface.

## Quick Start

### Start the raido service

```bash
proxy ❯❯ raido --help      # help options
proxy ❯❯ raido service run # for foreground mode
```

Or you can install raido as daemon and start it.

```bash
proxy ❯❯ raido service install   # install raido.service
proxy ❯❯ raido service start     # start raido in daemon mode
proxy ❯❯ raido service status    # check raido.service status
proxy ❯❯ raido service uninstall # uninstall raido.service
```

### Start the raido proxy server

```bash
proxy ❯❯ raido proxy start # start proxy server by default on address 0.0.0.0:8787
INF proxy started with cert hash: 5AE8BB04B096A6913A4EA45C35537355B82DB66DE40E201681F111CCDED73FFB
```

### Start agent on remote server

```bash
agent ❯❯ agent -pa 10.1.0.3:8787 -ch 5AE8BB04B096A6913A4EA45C35537355B82DB66DE40E201681F111CCDED73FFB
```

### Check all connected agents

```bash
proxy ❯❯ raido agent list # print all agents and their available routes in a table
┌────────────────────────┬───────────────────┬─────────────┬──────────┐
│           ID           │     Hostname      │   Routes    │  Status  │
├────────────────────────┼───────────────────┼─────────────┼──────────┤
│ LMD6Ycek8Rz6pXxL4kzLM8 │ root@1e28e066f43a │ 10.2.0.3/16 │ Inactive │
│                        │                   │ 10.3.0.3/16 │          │
│                        │                   │ 10.4.0.3/16 │          │
└────────────────────────┴───────────────────┴─────────────┴──────────┘
```

### Start tunneling to agent

```bash
proxy ❯❯ raido agent start-tunnel --agent-id LMD6Ycek8Rz6pXxL4kzLM8 # the command creates the tun interface and adds all routes
┌────────────────────────┬───────────────────┬─────────────┬──────────┐
│           ID           │     Hostname      │   Routes    │  Status  │
├────────────────────────┼───────────────────┼─────────────┼──────────┤
│ LMD6Ycek8Rz6pXxL4kzLM8 │ root@1e28e066f43a │ 10.2.0.3/16 │ Active   │
│                        │                   │ 10.3.0.3/16 │          │
│                        │                   │ 10.4.0.3/16 │          │
└────────────────────────┴───────────────────┴─────────────┴──────────┘
```

That's it, now you can send requests directly to these addresses.

## TODO

- Think about a way to transmit ICMP packets without changing the gVisor code. (Maybe use agent to detect hosts using icmp-echo requests) ¯\\_(ツ)_/¯
- Add new transport protocols for traffic tunneling
- Add the ability to build chains of agents
- Add the ability to independently select which addresses to add for tunneling
- Add the ability to stop and resume the tunnel
- Add multiplatform support
- Add logging options
- FIX BUGS!
