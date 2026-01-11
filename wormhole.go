package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dyastin-0/wormhole/core"
	wclient "github.com/Dyastin-0/wormhole/core/client"
	"github.com/Dyastin-0/wormhole/core/server"
	wserver "github.com/Dyastin-0/wormhole/core/server"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/common-nighthawk/go-figure"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"

	nethttp "net/http"
	_ "net/http/pprof"
)

var (
	ErrMissingID      = errors.New("missing id")
	ErrMissingTarget  = errors.New("missing target")
	ErrMissingAddress = errors.New("missing address")
)

func main() {
	w := New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	if err := w.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}

func New() *cli.Command {
	return &cli.Command{
		Name:    "wormhole",
		Usage:   "a tcp-based reverse tunnel service",
		Version: core.VERSION,
		Action:  wormholeCommand,
		Commands: []*cli.Command{
			startCommand(),
			httpCommand(),
			tcpCommand(),
			adminCommand(),
		},
	}
}

func wormholeCommand(ctx context.Context, cmd *cli.Command) error {
	figure := figure.NewFigure("wormhole-cli", "", true)
	figure.Print()

	fmt.Println()

	err := cli.ShowAppHelp(cmd)
	if err != nil {
		panic(err)
	}

	return nil
}

func startCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "start a wormhole server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Aliases: []string{"a", "addr"},
				Value:   ":8888",
				Usage:   "set the address where wormhole server will run",
			},
			&cli.StringFlag{
				Name:    "serveAddress",
				Aliases: []string{"sa", "serveAddr"},
				Value:   ":8889",
				Usage:   "set the address where wormhole tunnel handler will run",
			},
			&cli.StringFlag{
				Name:     "zoneID",
				Aliases:  []string{"z"},
				Usage:    "set cloudflare zone",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    "set cloudflare api token",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "domain",
				Aliases:  []string{"d"},
				Usage:    "set base domain for tunnels",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "ipv4",
				Aliases:  []string{"ip"},
				Usage:    "set ipv4 target for dns",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "pprof",
				Usage: "run wormhole with pprof",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "pprofAddr",
				Usage: "address used for pprof",
				Value: ":7060",
			},
		},
		Action: start,
	}
}

func start(ctx context.Context, cmd *cli.Command) error {
	addr := cmd.String("addr")
	serveAddr := cmd.String("serveAddr")
	zoneID := cmd.String("zoneID")
	token := cmd.String("token")
	domain := cmd.String("domain")
	ipV4 := cmd.String("ipv4")
	pprofAddr := cmd.String("pprofAddr")
	runPprof := cmd.Bool("pprof")

	if runPprof {
		go nethttp.ListenAndServe(pprofAddr, nil)
	}

	dnsManager, err := dnsmanager.NewCloudflare(
		dnsmanager.WithBaseDomain(domain),
		dnsmanager.WithToken(token),
		dnsmanager.WithZoneID(zoneID),
		dnsmanager.WithIPv4(ipV4),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize dns manager: %w", err)
	}

	secret := os.Getenv("WORMHOLE_SECRET")
	if secret == "" {
		return fmt.Errorf("secret is not set")
	}

	apiKeyIssuer, err := server.NewAPIKeyIssuer([]byte(secret))
	if err != nil {
		return err
	}

	wormholeServer, err := wserver.New(
		wserver.WithAddr(addr),
		wserver.WithServeAddr(serveAddr),
		wserver.WithDNSManager(dnsManager),
		wserver.WithAPIKeyIssuer(apiKeyIssuer),
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return wormholeServer.Run(gCtx)
	})

	g.Go(func() error {
		return wormholeServer.RunTunneler(gCtx)
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func httpCommand() *cli.Command {
	return &cli.Command{
		Name:   "http",
		Usage:  "start a wormhole http reverse tunnel client",
		Flags:  baseClientFlags(),
		Action: http,
	}
}

func http(ctx context.Context, cmd *cli.Command) error {
	name := cmd.String("name")
	addr := cmd.String("addr")
	targetAddr := cmd.String("targetAddr")
	apiKey := cmd.String("apiKey")
	ttl := cmd.Uint64("ttl")
	metrics := cmd.Bool("metrics")

	wormholeClient, err := wclient.New(
		wclient.WithProtoHTTP,
		wclient.WithName(name),
		wclient.WithAddr(addr),
		wclient.WithTargetAddr(targetAddr),
		wclient.WithMetrics(metrics),
		wclient.WithAPIKey(apiKey),
		wclient.WithTTL(ttl),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize wormhole client: %w", err)
	}

	err = wormholeClient.Run(ctx)
	if err != nil {
		return err
	}

	return nil
}

func tcpCommand() *cli.Command {
	return &cli.Command{
		Name:   "tcp",
		Usage:  "start a wormhole tcp reverse tunnel client",
		Flags:  baseClientFlags(),
		Action: tcp,
	}
}

func tcp(ctx context.Context, cmd *cli.Command) error {
	name := cmd.String("name")
	addr := cmd.String("addr")
	targetAddr := cmd.String("targetAddr")
	apiKey := cmd.String("apiKey")
	ttl := cmd.Uint64("ttl")
	metrics := cmd.Bool("metrics")

	wormholeClient, err := wclient.New(
		wclient.WithProtoTCP,
		wclient.WithName(name),
		wclient.WithAddr(addr),
		wclient.WithTargetAddr(targetAddr),
		wclient.WithMetrics(metrics),
		wclient.WithAPIKey(apiKey),
		wclient.WithTTL(ttl),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize wormhole client: %w", err)
	}

	err = wormholeClient.Run(ctx)
	if err != nil {
		return err
	}

	return nil
}

func baseClientFlags(flags ...cli.Flag) []cli.Flag {
	return append(
		flags,
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Usage:    "set your wormhole tunnel's domain (https://{name}.wormhole.dyastin.dev)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "targetAddress",
			Aliases:  []string{"targetAddr", "t"},
			Usage:    "set the address where connections will be tunneled to (eg., :3000)",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "address",
			Aliases: []string{"addr"},
			Usage:   "set the wormhole server address",
			Value:   "wormhole.dyastin.dev:443",
		},
		&cli.StringFlag{
			Name:    "apiKey",
			Aliases: []string{"k", "key"},
			Usage:   "API key token for authentication",
		},
		&cli.Uint64Flag{
			Name:  "ttl",
			Usage: "tunnel TTL in hours (only used with API key)",
			Value: 0,
		},
		&cli.BoolFlag{
			Name:    "metrics",
			Aliases: []string{"m"},
			Value:   false,
		},
	)
}

func adminCommand() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "administrative commands for wormhole server",
		Commands: []*cli.Command{
			issueTokenCommand(),
			generateSecretCommand(),
		},
	}
}

func issueTokenCommand() *cli.Command {
	return &cli.Command{
		Name:  "issue-token",
		Usage: "issue a new API key token",
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:     "ttl",
				Aliases:  []string{"t"},
				Usage:    "tunnel TTL in hours",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "expires",
				Aliases: []string{"e"},
				Usage:   "token expiration duration (e.g., 720h, 30d, 1y)",
				Value:   "2160h",
			},
		},
		Action: issueToken,
	}
}

func issueToken(ctx context.Context, cmd *cli.Command) error {
	secretStr := os.Getenv("WORMHOLE_SECRET")
	if secretStr == "" {
		return fmt.Errorf("secret not set")
	}

	secret, err := base64.StdEncoding.DecodeString(secretStr)
	if err != nil {
		return fmt.Errorf("failed to decode secret: %w", err)
	}

	issuer, err := server.NewAPIKeyIssuer(secret)
	if err != nil {
		return fmt.Errorf("failed to create issuer: %w", err)
	}

	ttl := cmd.Uint64("ttl")
	expiresStr := cmd.String("expires")

	expires, err := parseDuration(expiresStr)
	if err != nil {
		return fmt.Errorf("invalid expiration duration: %w", err)
	}

	token, err := issuer.Issue(ttl, expires)
	if err != nil {
		return fmt.Errorf("failed to issue token: %w", err)
	}

	fmt.Println("Token issued successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("TTL:          %d hours\n", ttl)
	fmt.Printf("Expires:      %s\n", time.Now().Add(expires).Format(time.RFC3339))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\nAPI Key:\n%s\n", token)
	fmt.Println("\nKeep this token secure! It cannot be recovered.")

	return nil
}

func generateSecretCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate-secret",
		Usage: "generate a new JWT signing secret",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "length",
				Aliases: []string{"l"},
				Usage:   "length of the secret in bytes",
				Value:   32,
			},
		},
		Action: generateSecret,
	}
}

func generateSecret(ctx context.Context, cmd *cli.Command) error {
	length := cmd.Int("length")
	if length < 16 {
		return fmt.Errorf("secret length must be at least 16 bytes")
	}

	secret := make([]byte, length)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("failed to generate secret: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(secret)

	fmt.Println("Secret generated successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Length:       %d bytes\n", length)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\nSecret (base64):\n%s\n", encoded)
	fmt.Println("\nStore this securely! Set as WORMHOLE_SECRET environment variable.")
	fmt.Printf("\nexport WORMHOLE_SECRET=\"%s\"\n", encoded)

	return nil
}

// parseDuration parses duration strings like "90d", "720h", "1y"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return time.ParseDuration(s)
	}

	switch unit {
	case 'd', 'D':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w', 'W':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'y', 'Y':
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	case 'h', 'H':
		return time.Duration(value) * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}
