package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Dyastin-0/wormhole"
)

func main() {
	ctx := context.Background()

	// start the local service
	go func() {
		localMux := http.NewServeMux()
		localMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello from the other end of a wormhole!")
		})

		log.Println("Local service listening at :8080")
		log.Fatal(http.ListenAndServe(":8080", localMux))
	}()

	// start the wormhole server
	go func() {
		w := wormhole.New(":3000", ctx)

		go func() {
			log.Println("Wormhole TCP server listening at :3000")
			log.Fatal(w.Start())
		}()

		// public http entrypoint
		mux := http.NewServeMux()
		mux.HandleFunc("/", w.HTTP)
		log.Println("Wormhole HTTP handler listening at :3001")
		log.Fatal(http.ListenAndServe(":3001", mux))
	}()

	// Start the wormhole client
	go func() {
		time.Sleep(1 * time.Second)

		c := wormhole.NewClient(
			"myid",  // unique id
			":3000", // wormhole address
			"3002",  // local client address
			":8080", // target adress
			"http",  // protocol
		)

		log.Println("Wormhole client connecting to server")

		if err := c.Start(); err != nil {
			log.Fatalf("client error: %v", err)
		}
	}()

	// wait and send test request
	time.Sleep(3 * time.Second)

	resp, err := http.Get("http://localhost:3001?id=myid")
	if err != nil {
		log.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("==== Tunnel Response ====")
	resp.Write(log.Writer())
}
