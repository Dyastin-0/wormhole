package main

import (
	"context"
	"crypto/tls"
	dbsql "database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Dyastin-0/wormhole"
	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/Dyastin-0/wormhole/logger"
	"github.com/Dyastin-0/wormhole/token"
	"github.com/common-nighthawk/go-figure"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v3"
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
		Name:   "wormhole",
		Usage:  "a simple tcp-based reverse tunnel",
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
				Name:     "ipv4",
				Usage:    "set ipv4 target for dns",
				Required: true,
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

	w := wormhole.NewServer(addr, httpAddr)

	conn, err := dbsql.Open("sqlite3", "file:dev.db?_foreign_keys=on&_journal_mode=WAL&_cache=shared&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	queries := db.New(conn)
	newStore := store.New(queries)

	newLogger := logger.New()

	logPath, err := LogPath("server")
	if err != nil {
		return err
	}

	newLogger.InitMultiWriter(logPath)

	issuer := token.DefaultIssuer()

	manager := dnsmanager.NewCloudflareManager(api, zone, baseDNS, ipv4)

	w.Store = newStore
	w.Logger = newLogger
	w.Issuer = issuer
	w.DNSManager = manager

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
				Name:    "id",
				Aliases: []string{"i"},
				Usage:   "set the tunnel id which wormhole client will use",
			},
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "set your wormhole tunnel's domain (https://{name}.wormhole.dyastin.tech)",
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
			&cli.StringFlag{
				Name:  "api",
				Usage: "set the wormhole api key",
			},
		},
		Action: http,
	}
}

func http(ctx context.Context, cmd *cli.Command) error {
	api := cmd.String("api")
	name := cmd.String("name")
	id := cmd.String("id")
	target := cmd.String("target")
	wsa := cmd.String("wormhole-server-address")

	tlsconfig := &tls.Config{
		ServerName: "wormhole.dyastin.tech",
	}

	c := wormhole.NewClient(api, id, name, wsa, target, wormhole.ProtoHTTP)

	newLogger := logger.New()

	logPath, err := LogPath("client")
	if err != nil {
		return err
	}

	newLogger.Init(logPath)

	c.Logger = newLogger

	err = c.Start(ctx, tlsconfig)
	if err != nil {
		return err
	}

	return nil
}

func LogPath(base string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, "wormhole-logs", base, "logs")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "wormhole.log")
	return logPath, nil
}
