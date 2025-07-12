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
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

type Wormhole struct {
	addr              string
	httpAddr          string
	mu                sync.Mutex
	tunnels           map[string]*tunnel
	Manager           *Manager
	cancel            context.CancelFunc
	ctx               context.Context
	logger            Logger
	tunnelHTTPRequest func(stream net.Conn, wr http.ResponseWriter, r *http.Request) error
}

func New(addr, httpAddr string) *Wormhole {
	logger := NewLogger()
	logger.InitMultiWriter("wormhole", "/var/log/wormhole/wormhole.log")

	return &Wormhole{
		addr:              addr,
		httpAddr:          httpAddr,
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

	parentCtx, cancel := context.WithCancel(ctx)
	w.ctx = parentCtx
	w.cancel = cancel

	errch := make(chan error, 2)

	go func() {
		err := w.start()
		if err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("tcp server exited: " + err.Error())
			cancel()
			errch <- err
		} else {
			errch <- nil
		}
	}()

	go func() {
		err := w.StartHTTP()
		if err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("http server exited: " + err.Error())
			cancel()
			errch <- err
		} else {
			errch <- nil
		}
	}()

	var finalErr error
	for i := 0; i < 2; i++ {
		if err := <-errch; err != nil && finalErr == nil {
			finalErr = err
		}
	}

	return finalErr
}

func (w *Wormhole) start() error {
	listener, err := net.Listen("tcp", w.addr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToListenToTCP, err)
	}

	go func() {
		<-w.ctx.Done()
		w.logger.Info("context cancelled, closing tcp listener")
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

	record := &Record{
		Name:    fmt.Sprintf("%s.%s", msg.ID, w.Manager.API.BaseDNS()),
		Content: w.Manager.API.IPV4(),
		Type:    "A",
		TTL:     720,
		Proxied: false,
	}

	dnsRecord, err := w.Manager.API.CreateDNSRecord(w.ctx, time.Minute*30, record)
	if err != nil {
		w.logger.Error(err.Error())
		session.Close()
	}

	<-session.CloseChan()

	err = w.Manager.API.DeleteDNSRecord(w.ctx, dnsRecord.ID)
	if err != nil {
		w.logger.Error(err.Error())
	}

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
	id := r.Header.Get("X-Forwarded-Host")

	if id == "" {
		id = r.Header.Get("Host")
	}

	w.logger.Debug("host: " + id)

	w.mu.Lock()
	t, ok := w.tunnels[strings.Replace(id, fmt.Sprintf(".%s", w.Manager.API.BaseDNS()), "", 1)]
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

func (w *Wormhole) StartHTTP() error {
	server := &http.Server{
		Addr:    w.httpAddr,
		Handler: http.HandlerFunc(w.HTTP),
	}

	go func() {
		<-w.ctx.Done()

		w.logger.Info("context cancelled, shutting down http server")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			w.logger.Error(fmt.Sprintf("%s: %v", fmt.Errorf("http server shutdown failed"), err))
		}
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("%w: %v", fmt.Errorf("http server stopped"), err)
	}

	return nil
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
