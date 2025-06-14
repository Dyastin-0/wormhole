package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/Dyastin-0/wormhole"
	"github.com/go-chi/chi/v5"
)

func main() {
	wormholeAddr := flag.String("wormholeAddr", ":8888", "TCP address to listen for wormhole connections")
	httpAddr := flag.String("httpAddr", ":8889", "HTTP address for incoming HTTP requests")

	flag.Parse()

	ctx := context.Background()
	wh := wormhole.New(*wormholeAddr, ctx)

	go func() {
		router := chi.NewRouter()
		router.Get("/{id}/*", wh.HTTP)

		log.Printf("HTTP server listening on %s\n", *httpAddr)
		if err := http.ListenAndServe(*httpAddr, router); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Printf("Wormhole server listening on %s\n", *wormholeAddr)
	if err := wh.Start(); err != nil {
		log.Fatalf("Wormhole server error: %v", err)
	}
}
