package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Dyastin-0/wormhole"
)

func main() {
	// start the local service
	go func() {
		localMux := http.NewServeMux()
		localMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello from the other end of a wormhole!")
		})

		log.Println("Local service listening at :8080")
		log.Fatal(http.ListenAndServe(":8080", localMux))
	}()

	// Start the wormhole client
	go func() {
		time.Sleep(1 * time.Second)

		c := wormhole.NewClient(
			"wormhole-client",            // unique id
			"wormhole.dyastin.tech:8888", // wormhole address
			":3002",                      // local client address
			":8080",                      // target adress
			"http",                       // protocol
		)

		log.Println("Wormhole client connecting to server")

		if err := c.Start(); err != nil {
			log.Fatalf("client error: %v", err)
		}
	}()

	// wait and send test request
	time.Sleep(3 * time.Second)

	resp, err := http.Get("https://wormhole.dyastin.tech/wormhole-client")
	if err != nil {
		log.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("==== Tunnel Response ====")
	resp.Write(log.Writer())

	// block
	select {}
}
