package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Dyastin-0/wormhole"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/common-nighthawk/go-figure"
	"github.com/urfave/cli/v3"
)

var (
	ErrMissingID      = errors.New("missing id")
	ErrMissingTarget  = errors.New("missing target")
	ErrMissingAddress = errors.New("missing address")
)

func main() {
	w := New()

	if err := w.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func New() *cli.Command {
	return &cli.Command{
		Name:   "wormhole-cli",
		Usage:  "simple tcp reverse tunnel",
		Action: wormholeCommand,
		Commands: []*cli.Command{
			startCommand(),
			httpCommand(),
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
				Name:    "httpAdress",
				Aliases: []string{"ha", "httpAddr"},
				Value:   ":8889",
				Usage:   "set the address where wormhole http handler will run",
			},
			&cli.StringFlag{
				Name:     "zone",
				Aliases:  []string{"z"},
				Usage:    "set cloudflare zone",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "api",
				Usage:    "set cloudflare api",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "dns",
				Usage:    "set base dns for tunnels",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "ipv4",
				Usage: "set ipv4 target for dns",
			},
		},
		Action: start,
	}
}

func start(ctx context.Context, cmd *cli.Command) error {
	addr := cmd.String("addr")
	httpAddr := cmd.String("httpAddr")
	zone := cmd.String("zone")
	api := cmd.String("api")
	baseDNS := cmd.String("dns")
	ipv4 := cmd.String("ipv4")

	w := wormhole.New(addr, httpAddr)

	manager, err := dnsmanager.NewCloudflareManager(api, zone, baseDNS, ipv4)
	if err != nil {
		return err
	}

	w.Manager = manager

	err = w.Start(ctx)
	if err != nil {
		return err
	}

	return nil
}

func httpCommand() *cli.Command {
	return &cli.Command{
		Name:  "http",
		Usage: "start a wormhole http reverse tunnel client",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Aliases:  []string{"i"},
				Usage:    "set the wormhole client's endpoint (https://{id}.wormhole.dyastin.tech)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "target",
				Aliases:  []string{"t"},
				Usage:    "set the address where the request will be tunneled to (:3000)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "wormhole-server-address",
				Aliases: []string{"s", "ws", "wsa", "server"},
				Usage:   "set the wormhole server address",
				Value:   "wormhole.dyastin.tech:8443",
			},
		},
		Action: http,
	}
}

func http(ctx context.Context, cmd *cli.Command) error {
	id := cmd.String("id")
	target := cmd.String("target")
	wsa := cmd.String("wormhole-server-address")

	tlsconfig := &tls.Config{
		ServerName: "wormhole.dyastin.tech",
	}

	c := wormhole.NewClient(id, wsa, target, wormhole.ProtoHTTP, tlsconfig)

	err := c.Start(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
