package wormhole

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/hashicorp/yamux"
)

type Wormhole struct {
	addr    string
	mu      sync.Mutex
	ctx     context.Context
	clients map[string]*yamux.Session
}

func NewWormhole(addr string, ctx context.Context) *Wormhole {
	return &Wormhole{
		addr:    addr,
		clients: make(map[string]*yamux.Session),
		ctx:     ctx,
	}
}

func (w *Wormhole) Start() error {
	// maybe implement tls later on
	listener, err := net.Listen("tcp", w.addr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(ErrFailedToAcceptConn.Error())
			continue
		}

		go w.handleConn(conn)
	}
}

func (w *Wormhole) handleConn(conn net.Conn) error {
	defer conn.Close()

	session, err := yamux.Server(conn, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToCreateYamuxServer, err)
	}

	stream, err := session.Accept()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToAcceptConn, err)
	}

	msg, err := w.handshake(stream)
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.clients[msg.ID] = session
	w.mu.Unlock()

	log.Printf("%s connected\n", msg.ID)

	<-session.CloseChan()

	w.mu.Lock()
	delete(w.clients, msg.ID)
	w.mu.Unlock()

	log.Printf("%s disconnected\n", msg.ID)

	return nil
}

// implements simple handshake
func (w *Wormhole) handshake(stream net.Conn) (*message, error) {
	defer stream.Close()

	dec := json.NewDecoder(stream)
	enc := json.NewEncoder(stream)

	var msg message

	err := dec.Decode(&msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToDecodeMessage, err)
	}

	if _, exists := w.clients[msg.ID]; exists {
		return nil, fmt.Errorf("%w: %v", ErrHandshakeFailed, "id already used")
	}

	err = enc.Encode(&message{
		ID:     msg.ID,
		Status: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHandshakeFailed, ErrFailedToEncodeMessage)
	}

	return &msg, nil
}

// forward request to the client
func (w *Wormhole) forward(req http.Request, wr http.ResponseWriter) error {
	// forward http request to yamux session
	// wait for response
	// write the response to http.ResponseWriter

	return nil
}
