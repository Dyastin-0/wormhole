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

```bash
wormhole start --zoneID <zone-id> --token <api-token> --domain <base-domain-for-tunnels> --ipv4 <your-server-ip> --address <:port-number> --serveAddress <:port-number>
```

`--address` is used to run a `TCP` server that handles Wormhole `client` connections.

`--serveAddress` is used to run a `TLS` server that route connections to it's configured tunnel based on it's `SNI`.

Note that not all DNS manager requires the `zone ID` for their API, e.g. Digital Ocean. Also, the Wormhole server uses plain `TCP`, you'll need to use a reverse proxy with `TLS`, as the client dials with `TLS`. Lastly, make sure the `<base-domain>` as well as `*.<base-domain>` points to your server.

### Running a Tunnel

Creating a tunnel is fairly simple, you just need to specify your desired and valid sub-domain, target address, and the wormhole server address. If you don't specify the `--address`, by default, it will connect to my self-hosted Wormhole server.

```bash
wormhole http --name <desired-subdomain-name> --targetAddr <:port-number> --address <wormhole.server.address:443>
```

You can use both `tcp` and `http` command to tunnel an `HTTP` server. And optionally, you can run it with `-m` flag to see a live metrics of your tunnel.

## Demo

![Demo](snapshots/demo.gif)
