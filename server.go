// Package wormhole
package wormhole

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/Dyastin-0/wormhole/logger"
	"github.com/Dyastin-0/wormhole/token"
	"github.com/hashicorp/yamux"
)

type Wormhole struct {
	addr     string
	httpAddr string

	tunnels    sync.Map // stores k=string;v=*tunnel
	Store      *store.Store
	DNSManager *dnsmanager.Manager
	Issuer     *token.Issuer
	Logger     logger.Logger

	donech            chan bool
	cancel            context.CancelFunc
	ctx               context.Context
	tunnelHTTPRequest func(stream net.Conn, wr http.ResponseWriter, r *http.Request) error
}

func New(addr, httpAddr string) *Wormhole {
	return &Wormhole{
		addr:              addr,
		httpAddr:          httpAddr,
		tunnelHTTPRequest: tunnelHTTPRequest,
		Logger:            &logger.NoopLogger{},
		donech:            make(chan bool, 1),
	}
}

func (w *Wormhole) Stop() {
	w.donech <- true
}

func (w *Wormhole) Start(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	if w.DNSManager == nil {
		return ErrNilDNSManager
	}

	if w.Store == nil {
		return ErrNilStore
	}

	w.ctx, w.cancel = context.WithCancel(ctx)

	// run donech listener
	go func() {
		<-w.donech
		w.cancel()
	}()

	errch := make(chan error, 2)

	go func() {
		err := w.start()
		if err != nil && !errors.Is(err, context.Canceled) {
			w.Logger.Error("tcp server exited: " + err.Error())
			w.cancel()
			errch <- err
		} else {
			errch <- nil
		}
	}()

	go func() {
		err := w.StartHTTP()
		if err != nil && !errors.Is(err, context.Canceled) {
			w.Logger.Error("http server exited: " + err.Error())
			w.cancel()
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

	w.Logger.Info("waiting for cleanup...")
	time.Sleep(2 * time.Second)
	w.Logger.Info("clean up done!")

	return finalErr
}

func (w *Wormhole) start() error {
	listener, err := net.Listen("tcp", w.addr)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToListenToTCP, err)
	}

	go func() {
		<-w.ctx.Done()
		w.Logger.Info("context cancelled, closing tcp listener")
		listener.Close()
	}()

	w.Logger.Info("service started")

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

				w.Logger.Error(fmt.Sprintf("%s: %s", ErrFailedToAcceptConn.Error(), err))
				continue

			}
		}

		go func(c net.Conn) {
			if err := w.handleConn(c); err != nil {
				w.Logger.Error(fmt.Sprintf("%v\n", err))
			}
		}(conn)
	}
}

func (w *Wormhole) handleConn(conn net.Conn) error {
	session, err := yamux.Server(conn, nil)
	if err != nil {
		errf := fmt.Errorf("%v: %w", ErrFailedToCreateYamuxServer, err)
		w.Logger.Error(errf.Error())
		return errf
	}
	defer session.Close()

	stream, err := session.Accept()
	if err != nil {
		errf := fmt.Errorf("%v: %w", ErrFailedToAcceptConn, err)
		w.Logger.Error(errf.Error())
		return errf
	}

	dec := json.NewDecoder(io.LimitReader(stream, MaxJSONSize))
	enc := json.NewEncoder(stream)

	dec.DisallowUnknownFields()

	domain, proto, ipv4, ttl, err := w.handshake(enc, dec)
	if err != nil {
		w.Logger.Error(err.Error())
		return err
	}

	record := &dnsmanager.Record{
		Name:    domain,
		Content: ipv4,
		Type:    dnsmanager.RecordTypeA,
		TTL:     1,
		Proxied: false,
	}

	dnsRecord, err := w.DNSManager.API.CreateDNSRecord(w.ctx, ttl, record)
	if err != nil {
		w.Logger.Error(err.Error())
		return err
	}

	w.tunnels.Store(domain, &tunnel{proto: proto, session: session})

	ttlexpired := make(chan bool, 1)
	// handle tunnel ttl
	go func(stream net.Conn, enc *json.Encoder) {
		<-time.After(ttl)

		err := enc.Encode(&message{
			Message: MsgTunnelttlTimeout,
		})
		if err != nil {
			w.Logger.Error(fmt.Sprintf("%s: %s", ErrFailedToEncodeMessage.Error(), err.Error()))
		}
		stream.Close()

		ttlexpired <- true
	}(stream, enc)

	select {
	case <-session.CloseChan():
		w.Logger.Debug("session closed")
	case <-w.ctx.Done():
		w.Logger.Debug("context canceled")
	case <-ttlexpired:
		w.Logger.WithStr("tunnel", dnsRecord.Meta.Name).Debug("ttl expired")
	}

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = w.DNSManager.API.DeleteDNSRecord(cleanupCtx, dnsRecord.ID)
	if err != nil {
		w.Logger.Error(err.Error())
	}

	<-cleanupCtx.Done()

	w.tunnels.Delete(domain)
	return nil
}
