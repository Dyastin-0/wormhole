package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	nethttp "net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Dyastin-0/wormhole/core"
	wclient "github.com/Dyastin-0/wormhole/core/client"
	wserver "github.com/Dyastin-0/wormhole/core/server"
	"github.com/Dyastin-0/wormhole/observer"
	"github.com/common-nighthawk/go-figure"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	_ "net/http/pprof"
)

var (
	ErrMissingID      = errors.New("missing id")
	ErrMissingTarget  = errors.New("missing target")
	ErrMissingAddress = errors.New("missing address")
)

// If not using linux, config file path must be passed as a cli flag.
const DefaultConfigPath = "/etc/wormhole/config.yaml"

type Config struct {
	Secret          string `yaml:"secret"`
	Domain          string `yaml:"domain"`
	Address         string `yaml:"address"`
	ServeAddress    string `yaml:"serveAddress"`
	PprofAddress    string `yaml:"pprofAddress"`
	Pprof           bool   `yaml:"withPprof"`
	ObserverAddress string `yaml:"observerAddress"`
	Observer        bool   `yaml:"withObserver"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func getValue[T any](configVal T, envVar string, flagVal T, fieldName string) (T, error) {
	var zero T

	// For string type
	if str, ok := any(flagVal).(string); ok {
		if envVar != "" {
			return any(envVar).(T), nil
		}
		if str != "" && str != any(zero).(string) {
			return flagVal, nil
		}
		if cfgStr, ok := any(configVal).(string); ok && cfgStr != "" {
			return configVal, nil
		}
		return zero, fmt.Errorf("%s not found. Set via --%s flag, config file, or environment variable", fieldName, fieldName)
	}

	// For bool type
	if _, ok := any(flagVal).(bool); ok {
		if envVar != "" {
			boolVal, err := strconv.ParseBool(envVar)
			if err != nil {
				return zero, fmt.Errorf("invalid boolean value for %s: %w", fieldName, err)
			}
			return any(boolVal).(T), nil
		}
		if any(flagVal).(bool) != any(zero).(bool) {
			return flagVal, nil
		}
		return configVal, nil
	}

	return zero, fmt.Errorf("unsupported type for getValue")
}

func main() {
	w := New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		time.Sleep(time.Millisecond * 500)
		cancel()
	}()

	if err := w.Run(ctx, os.Args); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("wormhole [inf] exited")
			return
		}
		fmt.Printf("wormhole [err] %s\n", err.Error())
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
				Name:    "serve-address",
				Aliases: []string{"sa"},
				Value:   ":8889",
				Usage:   "set the address where wormhole tunnel handler will run",
			},
			&cli.StringFlag{
				Name:    "domain",
				Aliases: []string{"d"},
				Usage:   "set base domain for tunnels (can be set via config file or DOMAIN env var)",
			},
			&cli.BoolFlag{
				Name:  "with-pprof",
				Usage: "run wormhole with pprof",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "pprof-address",
				Usage: "address used for pprof",
				Value: ":7060",
			},
			&cli.StringFlag{
				Name:  "secret",
				Usage: "set the secret used for validating api keys",
			},
			&cli.StringFlag{
				Name:  "config-path",
				Usage: "wormhole config path (override if config is somewhere else or not using linux)",
				Value: DefaultConfigPath,
			},
			&cli.BoolFlag{
				Name:  "with-observer",
				Usage: "enable telemetry",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "observer-address",
				Usage: "address where telemetry will run (e.g., :9090)",
				Value: ":9090",
			},
		},
		Action: start,
	}
}

func start(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config-path")

	cfg, err := loadConfig(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg == nil {
		cfg = &Config{}
	}

	addr, err := getValue(cfg.Address, os.Getenv("ADDRESS"), cmd.String("address"), "address")
	if err != nil {
		return err
	}

	serveAddr, err := getValue(cfg.ServeAddress, os.Getenv("SERVE_ADDRESS"), cmd.String("serve-address"), "serve-address")
	if err != nil {
		return err
	}

	pprofAddr, err := getValue(cfg.PprofAddress, os.Getenv("PPROF_ADDRESS"), cmd.String("pprof-address"), "pprof-address")
	if err != nil {
		return err
	}

	observerAddr, err := getValue(cfg.ObserverAddress, os.Getenv("OBSERVER_ADDRESS"), cmd.String("observer-address"), "observer-address")
	if err != nil {
		return err
	}

	runPprof, err := getValue(cfg.Pprof, os.Getenv("WITH_PPROF"), cmd.Bool("with-pprof"), "with-pprof")
	if err != nil {
		return err
	}

	runObserver, err := getValue(cfg.Observer, os.Getenv("WITH_OBSERVER"), cmd.Bool("with-observer"), "with-observer")
	if err != nil {
		return err
	}

	secretStr, err := getValue(cfg.Secret, os.Getenv("SECRET"), cmd.String("secret"), "secret")
	if err != nil {
		return err
	}

	secret, err := base64.StdEncoding.DecodeString(secretStr)
	if err != nil {
		return fmt.Errorf("failed to decode secret: %w", err)
	}

	apiKeyIssuer, err := wserver.NewAPIKeyIssuer(secret)
	if err != nil {
		return err
	}

	domain, err := getValue(cfg.Domain, os.Getenv("DOMAIN"), cmd.String("domain"), "domain")
	if err != nil {
		return err
	}

	// Setup observer
	var newObserver observer.Observer
	if runObserver {
		fmt.Printf("wormhole [inf] metrics enabled on %s\n", observerAddr)
		newObserver = observer.NewPrometheusObserver(prometheus.DefaultRegisterer)
	} else {
		newObserver = &observer.NoopObserver{}
	}

	wormholeServer, err := wserver.New(
		wserver.WithAddr(addr),
		wserver.WithServeAddr(serveAddr),
		wserver.WithDomain(domain),
		wserver.WithAPIKeyIssuer(apiKeyIssuer),
		wserver.WithObserver(newObserver),
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	if runPprof {
		fmt.Printf("wormhole [inf] pprof enabled on %s\n", pprofAddr)
		pprofServer := &nethttp.Server{
			Addr: pprofAddr,
		}

		g.Go(func() error {
			if err := pprofServer.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
				return err
			}
			return nil
		})

		g.Go(func() error {
			<-gCtx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			return pprofServer.Shutdown(shutdownCtx)
		})
	}

	if runObserver {
		g.Go(func() error {
			return wormholeServer.RunObserver(ctx, observerAddr)
		})
	}

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
	addr := cmd.String("address")
	targetAddr := cmd.String("target-address")
	apiKey := cmd.String("api-key")
	ttl := cmd.Uint64("ttl")
	metrics := cmd.Bool("metrics")
	httpLog := cmd.Bool("http-log")

	authType := cmd.String("auth-type")
	authUser := cmd.String("auth-user")
	authPass := cmd.String("auth-password")
	authToken := cmd.String("auth-token")

	opts := []wclient.OptFunc{
		wclient.WithProtoHTTP,
		wclient.WithName(name),
		wclient.WithAddr(addr),
		wclient.WithTargetAddr(targetAddr),
		wclient.WithMetrics(metrics),
		wclient.WithHTTPLog(httpLog),
		wclient.WithAPIKey(apiKey),
		wclient.WithTTL(ttl),
	}

	if err := addAuthOptions(&opts, authType, authUser, authPass, authToken); err != nil {
		return err
	}

	wormholeClient, err := wclient.New(opts...)
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
		Name:  "tcp",
		Usage: "start a wormhole tcp reverse tunnel client",
		Flags: append(baseClientFlags(),
			&cli.BoolFlag{
				Name:  "allow-http",
				Usage: "allow HTTP traffic on this TCP tunnel",
			}),
		Action: tcp,
	}
}

func tcp(ctx context.Context, cmd *cli.Command) error {
	name := cmd.String("name")
	addr := cmd.String("address")
	targetAddr := cmd.String("target-address")
	apiKey := cmd.String("api-key")
	ttl := cmd.Uint64("ttl")
	metrics := cmd.Bool("metrics")
	httpLog := cmd.Bool("http-log")
	allowHTTP := cmd.Bool("allow-http")

	authType := cmd.String("auth-type")
	authUser := cmd.String("auth-user")
	authPass := cmd.String("auth-password")
	authToken := cmd.String("auth-token")

	opts := []wclient.OptFunc{
		wclient.WithProtoTCP,
		wclient.WithName(name),
		wclient.WithAddr(addr),
		wclient.WithTargetAddr(targetAddr),
		wclient.WithMetrics(metrics),
		wclient.WithHTTPLog(httpLog),
		wclient.WithAPIKey(apiKey),
		wclient.WithTTL(ttl),
		wclient.WithAllowHTTP(allowHTTP),
	}

	if err := addAuthOptions(&opts, authType, authUser, authPass, authToken); err != nil {
		return err
	}

	wormholeClient, err := wclient.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to initialize wormhole client: %w", err)
	}

	err = wormholeClient.Run(ctx)
	if err != nil {
		return err
	}

	return nil
}

func addAuthOptions(opts *[]wclient.OptFunc, authType, authUser, authPass, authToken string) error {
	switch authType {
	case "basic":
		if authUser == "" || authPass == "" {
			return fmt.Errorf("basic auth requires both --auth-user and --auth-pass")
		}
		*opts = append(*opts, wclient.WithBasicAuth(authUser, authPass))
	case "bearer":
		if authToken == "" {
			return fmt.Errorf("bearer auth requires --auth-token")
		}
		*opts = append(*opts, wclient.WithBearerAuth(authToken))
	case "none", "":
		*opts = append(*opts, wclient.WithNoAuth)
	default:
		return fmt.Errorf("invalid auth type: %s (valid options: basic, bearer, none)", authType)
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
			Name:     "target-address",
			Aliases:  []string{"t"},
			Usage:    "set the address where connections will be tunneled to (eg., :3000)",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "address",
			Aliases: []string{"a"},
			Usage:   "set the wormhole server address",
			Value:   "wormhole.dyastin.dev:443",
		},
		&cli.StringFlag{
			Name:    "api-key",
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
			Usage:   "enable metrics streaming",
			Aliases: []string{"m"},
			Value:   false,
		},
		&cli.BoolFlag{
			Name:    "http-log",
			Usage:   "enable HTTP request logging (works with HTTP tunnels and TCP tunnels with --allow-http)",
			Aliases: []string{"hl"},
			Value:   false,
		},
		&cli.StringFlag{
			Name:  "auth-type",
			Usage: "authentication type: basic, bearer, or none (default: none)",
			Value: "none",
		},
		&cli.StringFlag{
			Name:    "auth-user",
			Aliases: []string{"u"},
			Usage:   "username for basic authentication",
		},
		&cli.StringFlag{
			Name:    "auth-password",
			Aliases: []string{"p"},
			Usage:   "password for basic authentication",
		},
		&cli.StringFlag{
			Name:  "auth-token",
			Usage: "bearer token for bearer authentication",
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
			&cli.StringFlag{
				Name:  "config-path",
				Usage: "wormhole config path (override if config is somewhere else or not using linux)",
				Value: DefaultConfigPath,
			},
		},
		Action: issueToken,
	}
}

func issueToken(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config-path")

	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &Config{}
	}

	secretStr, err := getValue(cfg.Secret, os.Getenv("SECRET"), cmd.String("secret"), "secret")
	if err != nil {
		return err
	}

	secret, err := base64.StdEncoding.DecodeString(secretStr)
	if err != nil {
		return fmt.Errorf("failed to decode secret: %w", err)
	}

	issuer, err := wserver.NewAPIKeyIssuer(secret)
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
	fmt.Println("\nStore this securely! Set as SECRET environment variable.")

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
