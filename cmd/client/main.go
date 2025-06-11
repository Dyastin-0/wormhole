package main

import (
	"log"

	"github.com/Dyastin-0/wormhole"
)

func main() {
	c := wormhole.NewClient("shesh", ":3000", ":3001", ":3002")

	log.Panic(c.Start())
}
