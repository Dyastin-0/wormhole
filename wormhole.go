package wormhole

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/yamux"
)

type Wormhole struct {
	addr              string
	mu                sync.Mutex
	tunnels           map[string]*tunnel
	cancel            context.CancelFunc
	ctx               context.Context
	logger            Logger
	tunnelHTTPRequest func(stream net.Conn, wr http.ResponseWriter, r *http.Request) error
}

func New(addr string) *Wormhole {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("could not determine home directory for logging")
	}

	name := "server"
	logPath := filepath.Join(home, "wormhole", name, "logs", "log.txt")

	logger := NewLogger()
	logger.InitMultiWriter(name, logPath)

	return &Wormhole{
		addr:              addr,
		tunnels:           make(map[string]*tunnel),
		logger:            logger,
		tunnelHTTPRequest: tunnelHTTPRequest,
	}
}

func (w *Wormhole) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *Wormhole) Start(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	w.ctx, w.cancel = context.WithCancel(ctx)

	listener, err := net.Listen("tcp", w.addr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToListenToTCP, err)
	}

	go func() {
		<-w.ctx.Done()
		listener.Close()
	}()

	w.logger.Info("service started")

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-w.ctx.Done():
				return w.ctx.Err()
			default:
				if errors.Is(err, net.ErrClosed) {
					return nil
				}

				w.logger.Error(fmt.Sprintf("%s: %s", ErrFailedToAcceptConn.Error(), err))
				continue

			}
		}

		go func(c net.Conn) {
			defer c.Close()

			if err := w.handleConn(c); err != nil {
				w.logger.Error(fmt.Sprintf("%v\n", err))
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

	if msg.Proto != ProtoHTTP && msg.Proto != ProtoTCP {
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

func (w *Wormhole) HTTP(wr http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	w.mu.Lock()
	t, ok := w.tunnels[id]
	w.mu.Unlock()

	if !ok {
		http.Error(wr, "tunnel not found", http.StatusNotFound)
		return
	}

	stream, err := t.session.Open()
	if err != nil {
		w.logger.Error(fmt.Sprintf("%s: %s\n", id, ErrFailedToOpenStream))
		http.Error(wr, "failed to open stream", http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	err = w.tunnelHTTPRequest(stream, wr, r)
	if err != nil {
		errf := fmt.Sprintf("tunnel error: %s", err.Error())

		w.logger.Error(errf)
		http.Error(wr, errf, http.StatusInternalServerError)
	}
}

func tunnelHTTPRequest(stream net.Conn, wr http.ResponseWriter, r *http.Request) error {
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
