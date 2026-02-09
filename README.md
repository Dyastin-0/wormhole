![gopher](/snapshots/gopher-getting-suck-in-a-wormhole.png)

# Wormhole

Expose local services to the internet through secure tunnels. Wormhole includes a built-in Terminal User Interface (TUI), WebSocket support, and end-to-end encryption.

The tool uses a single multiplexed ([yamux](https://github.com/hashicorp/yamux)) TCP stream to handle multiple virtual streams over a single connection.

---

## Features

* **Protocols**: HTTP/HTTPS, WebSocket, and raw TCP/TLS.
* **TUI**: Real-time traffic inspection and metrics dashboard.
* **Security**: TLS encryption, optional passthrough, and Basic/Bearer authentication.
* **Domains**: Custom CNAME support and automated wildcard SSL via CertMagic.
* **Observability**: Prometheus and OpenTelemetry (OTel) metrics/tracing.

---

## Installation

### Requirements

* Go 1.24 or later

### From Source

```bash
git clone https://github.com/Dyastin-0/wormhole.git
cd wormhole
go install .

```

---

## Tunnel Modes

### 1. HTTP and WebSocket

Terminates TLS at the server. Allows for inspection, logging, and authentication.

```bash
wormhole http --name myapp --target-address :3000

```

### 2. TCP with TLS Passthrough

Required for custom CNAMEs or when your local service manages its own SSL.

```bash
wormhole tcp --name api --url api.customdomain.com --target-address :443 --tls-passthrough

```

> **Note:** Only applicable on TCP tunnels with `--tls-passthrough`.

### 3. TCP Plaintext

Optimized for game servers (e.g., Minecraft) where low overhead is prioritized over encryption.

```bash
wormhole tcp --name mc-server --target-address :25565 --without-tls

```

---

## Monitoring and TUI

Launch the TUI by adding the `--metrics` or `--http-log` flags to any client command.

* **Bandwidth**: Real-time ingress/egress rates.
* **Inspection**: View headers and body content (ASCII/Hex).

![metrics](/snapshots/tui_metrics.png)

![inspect](/snapshots/tui_inspect.png)

---

## Tunnel Security

### Encryption

Traffic is encrypted via TLS between client and server by default.

* Use `--tls-passthrough` for end-to-end encryption to your local service.
* Use `--without-tls` to omit encryption (TCP only).
* HTTP tunnels are always TLS encrypted and terminate at the server.

### Authentication

Applies to HTTP tunnels or TCP tunnels with `--allow-http`.

* **Basic Auth**: `--auth-type basic --auth-user <user> --auth-password <pass>`
* **Bearer Token**: `--auth-type bearer --auth-token <token>`

---

## API Key Management

### Issue a Token

```bash
wormhole admin issue-token --ttl 24 --expires 30d

```

* **Flexible TTL** (`--ttl 0`): Client defines the duration.
* **Strict TTL** (`--ttl > 0`): Enforces a specific duration.

### Usage

```bash
wormhole tcp --name secure-app --api-key <token> --ttl 24

```

---

## Server Deployment

Running a Wormhole server requires a wildcard domain (e.g., `*.wormhole.dev`). By default, the server uses [certmagic](https://github.com/caddyserver/certmagic) for TLS certificates.

### Core Services

* **Control endpoint** (default `:8881`): Handles client-to-server signaling.
* **Tunnel endpoint** (default `:8889`): Routes public traffic based on SNI.

### Configuration

Configuration follows this precedence: **CLI flags > Environment variables > Config file**.

#### Environment Variables

Recommended for production. Reference `.example.env` for all variables.

```ini
# Systemd example
EnvironmentFile=/path/to/wormhole/.env

```

#### Command Line Flags

```bash
wormhole start \
    --secret <api-key-secret> \
    --domain <base-domain> \
    --address :8881 \
    --serve-address :8889

```

**Observability Flags:**

* `--with-observer`: Enable telemetry collection.
* `--observer`: Backend to use (`prom` or `otel`).
* `--observer-address`: Metrics endpoint address (default: `:9090`).
* `--with-tracer`: Enable OTel distributed tracing.

---

## Demo

![demo](/snapshots/demo.gif)

![minecraft](/snapshots/minecraft_demo.gif)
