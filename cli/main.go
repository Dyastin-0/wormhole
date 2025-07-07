package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Dyastin-0/wormhole"
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
				Usage:   "set the address where wormhole server will run",
			},
			&cli.StringFlag{
				Name:    "httpAdress",
				Aliases: []string{"ha", "httpAddr"},
				Usage:   "set the address where wormhole http handler will run",
			},
			&cli.StringFlag{
				Name:    "zone",
				Aliases: []string{"z"},
				Usage:   "set cloudflare zone",
			},
		},
		Action: start,
	}
}

func start(ctx context.Context, cmd *cli.Command) error {
	addr := cmd.String("addr")
	httpAddr := cmd.String("httpAddr")
	zone := cmd.String("zone")

	s := wormhole.New(addr, httpAddr, zone)

	err := s.Start(ctx)
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
				Name:    "id",
				Aliases: []string{"i"},
				Usage:   "set the wormhole client's endpoint (https://wormhole.dyastin.tech/{id})",
			},
			&cli.StringFlag{
				Name:    "address",
				Aliases: []string{"a", "addr"},
				Value:   ":8888",
				Usage:   "set the address where the wormhole client will run, default is :8888",
			},
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"t"},
				Usage:   "set the address where the request will be tunneled to (:3000)",
			},
			&cli.StringFlag{
				Name:    "wormhole-server-address",
				Aliases: []string{"s", "ws", "wsa", "server"},
				Usage:   "set the wormhole server address",
				Value:   "wormhole.dyastin.tech:8888",
			},
		},
		Action: http,
	}
}

func http(ctx context.Context, cmd *cli.Command) error {
	id := cmd.String("id")
	if id == "" {
		return ErrMissingID
	}

	addr := cmd.String("addr")
	if addr == "" {
		return ErrMissingAddress
	}

	target := cmd.String("target")
	if target == "" {
		return ErrMissingTarget
	}

	wsa := cmd.String("wormhole-server-address")

	c := wormhole.NewClient(id, wsa, addr, target, wormhole.ProtoHTTP)

	err := c.Start(ctx)
	if err != nil {
		return err
	}

	return nil
}
