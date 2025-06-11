package main

import (
	"context"
	"log"

	"github.com/Dyastin-0/wormhole"
)

func main() {
	ctx := context.Background()

	w := wormhole.NewWormhole(":3000", ctx)

	log.Println("wormhole running on port:3000")
	log.Fatal(w.Start())
}
