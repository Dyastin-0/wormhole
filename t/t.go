package main

import (
	"crypto/tls"
	"fmt"
	"log"
)

func main() {
	serverName := "wormhole.dyastin.tech"

	conn, err := tls.Dial("tcp", serverName+":8443", &tls.Config{
		ServerName: serverName, // Enables SNI and cert verification
	})
	if err != nil {
		log.Fatalf("TLS dial failed: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected securely to", serverName)

	_, err = conn.Write([]byte("hello\n"))
	if err != nil {
		log.Fatalf("write failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("read failed: %v", err)
	}

	fmt.Printf("Server replied: %s\n", string(buf[:n]))
}
