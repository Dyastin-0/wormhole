![gopher](/snapshots/gopher-getting-suck-in-a-wormhole.png)

# Wormhole

Expose your local services to the internet through secure tunnels. Perfect for webhooks, demos, and development.

Wormhole uses a single multiplexed ([yamux](https://github.com/hashicorp/yamux)) TCP stream.

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

## Server Deployment

Running a Wormhole server requires:
- A wildcard domain (e.g., `*.wormhole.dev`) pointed to your server
- The base domain (e.g., `wormhole.dev`) pointed to your server
- A reverse proxy that supports wildcard routing (nginx, Caddy, etc.)

Wormhole operates two core services:
- **Control endpoint** (default `:8881`): Handles client connections
- **Tunnel endpoint** (default `:8889`): Routes tunnel traffic based on SNI

### Configuration

Configuration can be loaded through multiple sources with the following precedence: CLI flags > environment variables > config file.

#### Environment Variables (Recommended)

The most secure approach for production deployments. Reference `.example.env` for all available variables.

For systemd services, inject your environment file:
```ini
EnvironmentFile=/path/to/wormhole/.env
```

#### Configuration File

Store a configuration file at the default path (`/etc/wormhole/config.yaml` on Linux) to automatically load settings. See `example.config.yaml` for reference.

#### Command Line Flags
```bash
wormhole start \
    --secret <api-key-secret> \
    --domain <base-domain-for-tunnels> \
    --address <:port-number> \
    --serve-address <:port-number>
```

**Available Flags:**

- `--address`: TCP server address for handling client connections (default: `:8881`)
- `--serve-address`: TLS server address that routes tunnel traffic based on SNI (default: `:8889`)
- `--secret`: Secret key used to generate and validate API keys (required)
- `--domain`: Base domain for tunnels (e.g., `wormhole.dev`)
- `--config-path`: Custom path to configuration file (default: `/etc/wormhole/config.yaml`)

**Observability and Profiling Options:**

- `--with-pprof`: Enable pprof profiling for performance analysis
- `--pprof-address`: Address for pprof endpoint (default: `:7060`)
- `--with-observer`: Enable telemetry and metrics collection
- `--observer`: Observer backend to use (`prom` or `otel`, default: `prom`)
- `--with-prom-exporter`: Use Prometheus exporter with the observer (applies when `--observer` is set to `otel`)
- `--observer-address`: Address where metrics endpoint will run (default: `:9090`)
- `--collector-address`: OpenTelemetry collector address, used when observer is set to `otel` (default: `:4327`, required when `--observer` is set to `otel`)
- `--with-tracer`: Enable distributed tracing with OpenTelemetry (uses Tempo, `--tempo-address` must be set)
- `--tempo-address`: Tempo endpoint for trace collection (default: `:4317`)

**Starting the Server:**

With environment variables or config file:
```bash
wormhole start
```

With custom config path:
```bash
wormhole start --config-path "/path/to/config.yaml"
```

With observability enabled:
```bash
wormhole start \
    --with-observer \
    --observer prom \
    --observer-address :9090
```

With OpenTelemetry and Prometheus exporter:
```bash
wormhole start \
    --with-observer \
    --observer otel \
    --with-prom-exporter \
    --observer-address :9090
```

With OpenTelemetry and collector:
```bash
wormhole start \
    --with-observer \
    --observer otel \
    --collector-address :4327
```

## Client Usage

### HTTP Tunnels

Create an HTTP tunnel by specifying your subdomain, target address, and server address:
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --address <wormhole.server.address:443>
```

The `--address` flag is optional and defaults to `wormhole.dyastin.dev:443`.

**Example:**
```bash
wormhole http \
    --name myapp \
    --target-address :3000
```

This creates a tunnel accessible at `https://myapp.wormhole.dyastin.dev` that forwards traffic to your local port 3000.

### TCP Tunnels

Create a raw TCP tunnel:
```bash
wormhole tcp \
    --name <subdomain> \
    --target-address <:port-number> \
    --address <wormhole.server.address:443>
```

By default, TCP tunnels block HTTP traffic. To allow HTTP traffic on a TCP tunnel, use the `--allow-http` flag:
```bash
wormhole tcp \
    --name <subdomain> \
    --target-address <:port-number> \
    --allow-http
```

This allows you to use HTTP-specific features like HTTP request logging and authentication with TCP tunnels.

## Tunnel Security

### Authentication

Authentication only applies to HTTP tunnels, or TCP tunnels with `--allow-http`.

#### Basic Authentication
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --auth-type basic \
    --auth-user <username> \
    --auth-password <password>
```

Clients will be prompted for credentials:
```bash
curl -u username:password https://<subdomain>.wormhole.dev
```

#### Bearer Token Authentication
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --auth-type bearer \
    --auth-token <your-secret-token>
```

Clients must include the token in the Authorization header:
```bash
curl -H "Authorization: Bearer <your-secret-token>" https://<subdomain>.wormhole.dev
```

#### Disable Authentication

Explicitly disable authentication (default behavior):
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --auth-type none
```

## Monitoring and Observability

### Real-time Metrics

Stream live tunnel performance metrics:
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --metrics
```

Displays:
- Ingress and egress bandwidth with transfer rates
- Active and total connection counts
- Tunnel uptime
- Round-trip time (RTT)

### HTTP Request Logging

Monitor all incoming HTTP requests in real-time:
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --http-log
```

Each request displays:
- Timestamp
- HTTP method (GET, POST, PUT, DELETE, etc.)
- Request path
- Response status code
- Response time in milliseconds

HTTP logging works with both HTTP tunnels and TCP tunnels that have `--allow-http` enabled:
```bash
wormhole tcp \
    --name <subdomain> \
    --target-address <:port-number> \
    --allow-http \
    --http-log
```

### Combined Monitoring

Enable comprehensive observability with both metrics and request logging:
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --metrics \
    --http-log
```

## API Key Management

### Generating a Secret

Generate a cryptographically secure secret for API key signing:
```bash
wormhole admin generate-secret --length 32
```

Store this secret securely and configure it via the `SECRET` environment variable or in your config file.

### Issuing API Keys

Issue API keys to grant clients extended tunnel lifetimes:
```bash
wormhole admin issue-token --ttl 4 --expires 30d
```

**Parameters:**

- `--ttl`: Time-to-live in hours for tunnels created with this key
- `--expires`: JWT expiration duration (e.g., `30d`, `720h`, `1y`, `52w`)

**TTL Behavior:**

- **Fixed TTL** (TTL > 0): All tunnels created with this key will have the specified TTL (e.g., `--ttl 4` creates 4-hour tunnels)
- **Flexible TTL** (TTL = 0): Clients can specify their own `--ttl` value up to a server-defined maximum
- **No API Key**: Tunnels default to 1-hour lifetimes without an API key

### Using API Keys

Create tunnels with extended lifetimes using your API key:
```bash
wormhole http \
    --name <subdomain> \
    --target-address <:port-number> \
    --api-key <api-key-token> \
    --ttl 24
```

If `--ttl` is omitted, the tunnel defaults to 1 hour unless the API key has a specific TTL claim.

## Complete Examples

### Authenticated HTTP Tunnel with Full Monitoring

Create a production-ready tunnel with authentication, metrics, and request logging:
```bash
wormhole http \
    --name myapp \
    --target-address :3000 \
    --address wormhole.dyastin.dev:443 \
    --api-key eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9... \
    --ttl 24 \
    --auth-type bearer \
    --auth-token my-secret-token-123 \
    --metrics \
    --http-log
```

Access the tunnel:
```bash
curl -H "Authorization: Bearer my-secret-token-123" https://myapp.wormhole.dyastin.dev
```

### TCP Tunnel with HTTP Support and Basic Auth

Create a flexible TCP tunnel that supports HTTP traffic with authentication:
```bash
wormhole tcp \
    --name secure-app \
    --target-address :8080 \
    --address wormhole.dyastin.dev:443 \
    --allow-http \
    --auth-type basic \
    --auth-user admin \
    --auth-password secret123 \
    --http-log
```

Access the tunnel:
```bash
curl -u admin:secret123 https://secure-app.wormhole.dyastin.dev
```

### Long-lived Tunnel with API Key

Create a tunnel that stays active for 7 days:
```bash
# First, issue a long-lived API key
wormhole admin issue-token --ttl 168 --expires 30d

# Then create the tunnel
wormhole http \
    --name longlived \
    --target-address :8080 \
    --api-key <issued-key> \
    --ttl 168
```

## Demo

![Demo](snapshots/demo.gif)
