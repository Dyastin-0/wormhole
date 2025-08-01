package main

import (
	"context"
	dbsql "database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/api/server"
	"github.com/Dyastin-0/wormhole/api/store"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	conn, err := dbsql.Open("sqlite3", "file:dev.db?_foreign_keys=on&_journal_mode=WAL&_cache=shared&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	queries := db.New(conn)
	newStore := store.New(queries)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	s := server.New(":42069", ":42070", newStore)

	err = s.Start(ctx)
	if err != nil {
		return
	}
}
