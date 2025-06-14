package main

import (
	"flag"
	"log"

	"github.com/Dyastin-0/wormhole"
)

func main() {
	id := flag.String("id", "", "unique identifier for the tunnel")
	wormholeAddr := flag.String("wormholeAddr", "localhost:8888", "wormhole server address")
	localAddr := flag.String("localAddr", ":9999", "local address to bind the wormhole client")
	targetAddr := flag.String("targetAddr", ":3000", "address of the target service to expose")
	proto := flag.String("proto", "http", "protocol to use (e.g., tcp)")

	flag.Parse()

	if *id == "" {
		log.Fatal("id is required")
	}

	client := wormhole.NewClient(*id, *wormholeAddr, *localAddr, *targetAddr, *proto)

	if err := client.Start(); err != nil {
		log.Fatalf("failed to start wormhole client: %v", err)
	}
}
