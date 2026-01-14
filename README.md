# Wormhole
A TCP-based reverse tunnel service written in `Go`.

## How to Install

### From Source

```bash
git clone https://github.com/Dyastin-0/wormhole.git
```

```bash
cd wormhole
```

```bash
go install .
```

### With Go

```bash
go install github.com/Dyastin-0/wormhole@latest
```

## How to Use

### Running a Server

Running a server requires a `domain` and an `API token` with `Edit` permission, at least for the specific `zone`, from a DNS manager, as it automatically manages DNS records for tunnels. Currently, there's an implementation of `DNSManager` with `Cloudflare`, you can implement the `DNSManager` interface with whichever DNS manager you are using.

There are multiple ways to load configs.

#### Environment (Recommended)

Safest way is to load them in the environment, see `.example.env`. If you are running the server as a service, you can inject your `.env` via:

```
EnvironmentFile=/your/wormhole/.env
```

#### Config File

You can store a configuration file at `DefaultConfigFile` (see `wormhole.go`) to automatically use them. See `example.config.yaml`.

#### CLI Flags

```bash
wormhole start \
    --secret <api-key-secret> \
    --zoneID <zone-id> \
    --token <api-token> \
    --domain <base-domain-for-tunnels> \
    --ipv4 <your-server-ip> \
    --address <:port-number> \
    --serveAddress <:port-number>
```

`--address` is used to run a `TCP` server that handles Wormhole `client` connections.

`--serveAddress` is used to run a `TLS` server that route connections to it's configured tunnel based on it's `SNI`.

Note that not all DNS manager requires the `zone ID` for their API, e.g. Digital Ocean. Also, the Wormhole server uses plain `TCP`, you'll need to use a reverse proxy with `TLS`, as the client dials with `TLS`. Lastly, make sure the `<base-domain>` as well as `*.<base-domain>` points to your server.

---

If you are using environment/config file, simply run:

```bash
wormhole start
```

Or if you put your configuration somewhere else:

```bash
wormhole start --configPath "/somewhere/somewhere/config.yaml"
```

### Running a Tunnel

Creating a tunnel is fairly simple, you just need to specify your desired and valid sub-domain, target address, and the wormhole server address. If you don't specify the `--address`, by default, it will connect to my self-hosted Wormhole server.

```bash
wormhole http \
    --name <desired-subdomain-name> \
    --targetAddr <:port-number> \
    --address <wormhole.server.address:443>
```

You can use both `tcp` and `http` command to tunnel an `HTTP` server. And optionally, you can run it with `-m` flag to see a live metrics of your tunnel.

### Generating an API Key

There's a special server side command, `admin` where it can issue API keys meant to be used by clients.

```bash
wormhole admin issue-token --expires 30d --ttl 4
```

`--expires` is used to indicate the jwt expiration.
`--ttl` is used to inject a custom claim `time-to-live` (in hours) on the JWT token, which is used by the client when requesting a tunnel.

Clients with an API key can have a longer tunnel `time-to-live`, depending on the `TTL` value injected into the claims.
If the `TTL` claims is zero (0), the server treats it differently. Clients with an API key that has zero (0) `TTL` on its claim can specify
a `--ttl` flag to customize the tunnel's `time-to-live` via the `--ttl` flag.

API key is specified as a flag, `--apiKey`.
If a client specified `--apiKey`, but has not specified `--ttl`, tunnel's `time-to-live` will use the default one (1) hour.

```bash
wormhole http \ 
    --name <desired-subdomain-name> \ 
    --targetAddr <:port-number> \
    --address <wormhole.server.address:443> \
    --apiKey <api-key> \
    --ttl 24 
```

Clients with no API key will always have a tunnel with one (1) hour `time-to-live`, but can still freely choose their own subdomain.

## Demo

![Demo](snapshots/demo.gif)
