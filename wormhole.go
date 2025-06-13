package wormhole

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	tunnels map[string]*tunnel
}

func New(addr string, ctx context.Context) *Wormhole {
	return &Wormhole{
		addr:    addr,
		tunnels: make(map[string]*tunnel),
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

		go func(c net.Conn) {
			if err := w.handleConn(c); err != nil {
				log.Printf("%v\n", err)
			}
		}(conn)
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
	w.tunnels[msg.ID] = &tunnel{proto: msg.Proto, session: session}
	w.mu.Unlock()

	<-session.CloseChan()

	w.mu.Lock()
	delete(w.tunnels, msg.ID)
	w.mu.Unlock()

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

	if _, exists := w.tunnels[msg.ID]; exists {
		errMsg := &message{ID: msg.ID, Status: 1, Err: ErrIDAlreadyUsed.Error()}

		err = enc.Encode(errMsg)
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf("%w: %v", ErrHandshakeFailed, ErrIDAlreadyUsed),
				fmt.Errorf("%w: %v", ErrFailedToEncodeMessage, err),
			)
		}

		return nil, fmt.Errorf("%w: %v", ErrHandshakeFailed, ErrIDAlreadyUsed)
	}

	if msg.Proto != httpProto && msg.Proto != tcpProto {
		errMsg := &message{ID: msg.ID, Status: 1, Err: ErrUnsupportedProtocol.Error()}

		err = enc.Encode(errMsg)
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf("%w: %v", ErrHandshakeFailed, ErrUnsupportedProtocol),
				fmt.Errorf("%w: %v", ErrFailedToEncodeMessage, err),
			)
		}

		return nil, ErrUnsupportedProtocol
	}

	err = enc.Encode(&message{
		ID:     msg.ID,
		Status: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHandshakeFailed, err)
	}

	return &msg, nil
}

func (w *Wormhole) http(stream net.Conn, wr http.ResponseWriter, r *http.Request) error {
	err := r.Write(stream)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToWriteHTTPTunnelRequest, err)
	}

	bufr := bufio.NewReader(stream)

	resp, err := http.ReadResponse(bufr, r)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToReadHTTPTunnelResponse, err)
	}

	defer resp.Body.Close()

	copyHeader(wr.Header(), resp.Header)
	io.Copy(wr, resp.Body)

	return nil
}

func (w *Wormhole) HTTP(wr http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	w.mu.Lock()
	t, ok := w.tunnels[id]
	w.mu.Unlock()

	if !ok {
		http.Error(wr, "tunnel not found", http.StatusNotFound)
		return
	}

	stream, err := t.session.Open()
	if err != nil {
		log.Printf("%s: %s\n", id, ErrFailedToOpenStream)
		http.Error(wr, "failed to open stream", http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	err = w.http(stream, wr, r)
	if err != nil {
		http.Error(wr, fmt.Sprintf("tunnel error: %s", err.Error()), http.StatusInternalServerError)
	}
}

func (w *Wormhole) tcp(stream net.Conn) error {
	return nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
