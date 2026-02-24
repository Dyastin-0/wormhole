![gopher](/snapshots/gopher-getting-suck-in-a-wormhole.png)

# Wormhole

Secure tunneling solution for exposing local services to the internet. Features a built-in Terminal User Interface ([bubbletea](https://github.com/charmbracelet/bubbletea)) and end-to-end encryption.

## Demos

![demo](/snapshots/demo.gif)

![minecraft](/snapshots/minecraft_demo.gif)

Wormhole uses [yamux](https://github.com/hashicorp/yamux) to tunnel multiple inbound connections over a single TCP stream.

---

## Features

- **Protocol Support**: HTTP/HTTPS, WebSocket, and raw TCP/TLS tunnels
- **TUI**: Real-time traffic inspection and metrics dashboard
- **Security**: TLS encryption, optional end-to-end passthrough, and Basic/Bearer authentication
- **Custom Domains**: CNAME support with automated wildcard TLS via [CertMagic](https://github.com/caddyserver/certmagic)
- **Observability**: Prometheus and OpenTelemetry metrics/tracing

---

You can read a [here](https://dyastin.dev/post/how-to-use-the-wormhole-cli) on how to use the CLI.
