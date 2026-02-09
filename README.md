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

## Installation

### Requirements
- Go 1.24 or later

### From Source
```bash
git clone https://github.com/Dyastin-0/wormhole.git
cd wormhole
go install .
```

### Using Go Install
```bash
go install github.com/Dyastin-0/wormhole@latest
```

---

## Quick Start

### HTTP Tunnel 
```bash
wormhole http --name myapp --target-address :3000
```

Access the tunnel:

```bash
curl https://myapp.wormhole.dyastin.dev
```

### WebSocket Support

WebSocket connections work automatically with HTTP and TCP tunnels.

### TCP Tunnel with TLS Passthrough

For custom domains or when your service handles its own TLS:

```bash
wormhole tcp --name api --url https://api.customdomain.com --target-address :443 --tls-passthrough
```

Make sure `--url` is configured on your DNS manager. If not yet, create a CNAME record and point it to the resulting domain (e.g., your.domain.com -> api.wormhole.dyastin.dev). 

### TCP Tunnel Without Encryption

Great for game servers (e.g., Minecraft) where low overhead matters:

```bash
wormhole tcp --name mc-server --target-address :25565 --without-tls
```

---

## Tunnel Modes

| Mode | Use Case | Encryption | Authentication | Inspection |
|------|----------|------------|----------------|------------|
| **HTTP** | Web apps, APIs, webhooks | TLS (server-terminated) | ✓ | ✓ |
| **TCP + `--allow-http`** | TCP with HTTP features | TLS (server-terminated)  | ✓ | ✓ |
| **TCP + TLS Passthrough** | Custom domains, end-to-end encryption | End-to-end TLS | ✗ | ✗ |
| **TCP Plain-text** | Game servers, low latency | None | ✗ | ✗ |

---

## Monitoring and TUI

Launch the interactive dashboard by adding `--metrics` or `--http-log` to any tunnel command:
```bash
wormhole http --name myapp --target-address :3000 --metrics --http-log
```

**Dashboard Features:**
- Real-time bandwidth (ingress/egress rates)
- Active and total connection counts
- HTTP request logging (method, path, status, timing)
- Traffic inspection (headers and body in ASCII/Hex)
- Round-trip time (RTT) monitoring

![metrics](/snapshots/tui_metrics.png)
![inspect](/snapshots/tui_inspect.png)

---

## Security

### Encryption Options

**Default**: All traffic is TLS-encrypted between client and server.

- **HTTP tunnels**: Always use TLS (terminated at server)
- **TCP with `--tls-passthrough`**: End-to-end encryption to your local service
- **TCP with `--without-tls`**: No encryption

### Authentication

Available for:
- **HTTP tunnels** (always)
- **TCP tunnels with `--allow-http`** (enables HTTP-layer features)

> **Note**: Plain-text TCP tunnels without `--allow-http` operate at the transport layer and cannot use HTTP-based authentication. Add `--allow-http` to enable HTTP-specific features like auth and request logging on TCP tunnels.

#### Basic Authentication
```bash
wormhole http \
    --name secure-app \
    --target-address :3000 \
    --auth-type basic \
    --auth-user admin \
    --auth-password secret123
```

Access the tunnel:
```bash
curl -u admin:secret123 https://secure-app.wormhole.dyastin.dev
```

#### Bearer Token
```bash
wormhole http \
    --name api \
    --target-address :8080 \
    --auth-type bearer \
    --auth-token my-secret-token
```

Access the tunnel:
```bash
curl -H "Authorization: Bearer my-secret-token" https://api.wormhole.dyastin.dev
```

#### TCP with HTTP Features and Authentication

```bash
wormhole tcp \
    --name secure-app \
    --target-address :8080 \
    --allow-http \
    --auth-type basic \
    --auth-user admin \
    --auth-password secret123 \
    --http-log
```
With plain-text:

```bash
wormhole tcp \
    --name secure-app \
    --target-address :8080 \
    --without-tls \
    --allow-http \
    --auth-type basic \
    --auth-user admin \
    --auth-password secret123 \
    --http-log
```

---

## API Key Management

API keys extend tunnel lifetimes beyond the default 1-hour limit.

### Generate a Secret (Server Setup)
```bash
wormhole admin generate-secret --length 32
```

Store this in your server configuration via `SECRET` environment variable or config file.

### Issue an API Key
```bash
wormhole admin issue-token --ttl 24 --expires 30d
```

**Parameters:**
- `--ttl > 0`: Fixed TTL—all tunnels created with this key automatically get this exact lifetime
- `--ttl 0`: Privileged key—client must specify `--ttl` when creating tunnels (up to server maximum)
- `--expires`: How long the API key itself remains valid (e.g., `30d`, `1y`, `720h`, `52w`)

**Default Behavior**: Without an API key, tunnels have a 1-hour lifetime.

### Use an API Key

**With a fixed TTL key (issued with `--ttl > 0`):**
```bash
# API key was issued with --ttl 24
wormhole http \
    --name myapp \
    --target-address :8080 \
    --api-key <your-token>
    # Tunnel automatically gets 24-hour lifetime
```

**With a privileged key (issued with `--ttl 0`):**
```bash
# API key was issued with --ttl 0
wormhole http \
    --name myapp \
    --target-address :8080 \
    --api-key <your-token> \
    --ttl 168  # Client specifies 7 days
```

---

## Server Deployment

### Requirements

- A wildcard domain (e.g., `*.wormhole.dev`) pointed to your server
- The base domain (e.g., `wormhole.dev`) pointed to your server

By default, Wormhole uses [CertMagic](https://github.com/caddyserver/certmagic) for automatic SSL certificate management.

### Core Services

- **Control endpoint** (default `:8881`): Handles client connections and signaling
- **Tunnel endpoint** (default `:8889`): Routes public traffic based on SNI

### Configuration Priority

**CLI flags > Environment variables > Config file**

#### Environment Variables (Recommended)

Reference `.example.env` for all available options.

For systemd services:
```ini
[Service]
EnvironmentFile=/path/to/wormhole/.env
```

#### Configuration File

Place your config at `/etc/wormhole/config.yaml` (Linux) for automatic loading. See `example.config.yaml` for reference.

#### Command Line
```bash
wormhole start \
    --secret <api-key-secret> \
    --domain <base-domain> \
    --address :8881 \
    --serve-address :8889
```

**Core Flags:**
- `--secret`: Secret for API key signing (required)
- `--domain`: Base domain for tunnels (e.g., `wormhole.dev`)
- `--address`: Control endpoint address (default: `:8881`)
- `--serve-address`: Tunnel endpoint address (default: `:8889`)
- `--config-path`: Custom config file path (default: `/etc/wormhole/config.yaml`)

**Observability Flags:**
- `--with-observer`: Enable metrics collection
- `--observer`: Backend (`prom` or `otel`, default: `prom`)
- `--observer-address`: Metrics endpoint (default: `:9090`)
- `--with-prom-exporter`: Use Prometheus exporter with OTel
- `--collector-address`: OTel collector address (default: `:4327`)
- `--with-tracer`: Enable distributed tracing
- `--tempo-address`: Tempo endpoint (default: `:4317`)

**Profiling:**
- `--with-pprof`: Enable performance profiling
- `--pprof-address`: Profiling endpoint (default: `:7060`)

### Example Deployments

**Basic:**
```bash
wormhole start
```

**With Prometheus metrics:**
```bash
wormhole start \
    --with-observer \
    --observer prom \
    --observer-address :9090
```

**With OpenTelemetry:**
```bash
wormhole start \
    --with-observer \
    --observer otel \
    --collector-address :4327
```

**With OpenTelemetry and Prometheus exporter:**
```bash
wormhole start \
    --with-observer \
    --observer otel \
    --with-prom-exporter \
    --observer-address :9090
```

---

## Complete Examples

### Production HTTP Tunnel
```bash
wormhole http \
    --name myapp \
    --target-address :3000 \
    --api-key <your-key> \
    --auth-type bearer \
    --auth-token my-secret-token \
    --metrics \
    --http-log
```

Access:
```bash
curl -H "Authorization: Bearer my-secret-token" https://myapp.wormhole.dyastin.dev
```

### TCP Tunnel with HTTP Features
```bash
wormhole tcp \
    --name secure-app \
    --target-address :8080 \
    --allow-http \
    --auth-type basic \
    --auth-user admin \
    --auth-password secret123 \
    --http-log
```

Access:
```bash
curl -u admin:secret123 https://secure-app.wormhole.dyastin.dev
```

### Custom Domain with End-to-End Encryption
```bash
wormhole tcp \
    --name api \
    --url api.yourdomain.com \
    --target-address :443 \
    --tls-passthrough \
    --api-key <your-key>
```

### Long-lived Tunnel with Privileged API Key
```bash
# First, issue a privileged API key
wormhole admin issue-token --ttl 0 --expires 30d

# Then create a 7-day tunnel
wormhole http \
    --name longlived \
    --target-address :8080 \
    --api-key <issued-key> \
    --ttl 168
```
