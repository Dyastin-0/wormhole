package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dyastin-0/wormhole/core"
	wclient "github.com/Dyastin-0/wormhole/core/client"
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

	wormholeServer, err := wserver.New(
		wserver.WithAddr(addr),
		wserver.WithServeAddr(serveAddr),
		wserver.WithDNSManager(dnsManager),
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
	metrics := cmd.Bool("metrics")

	wormholeClient, err := wclient.New(
		wclient.WithProtoHTTP,
		wclient.WithName(name),
		wclient.WithAddr(addr),
		wclient.WithTargetAddr(targetAddr),
		wclient.WithMetrics(metrics),
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
	metrics := cmd.Bool("metrics")

	wormholeClient, err := wclient.New(
		wclient.WithProtoTCP,
		wclient.WithName(name),
		wclient.WithAddr(addr),
		wclient.WithTargetAddr(targetAddr),
		wclient.WithMetrics(metrics),
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
		&cli.BoolFlag{
			Name:    "metrics",
			Aliases: []string{"m"},
			Value:   false,
		},
	)
}
