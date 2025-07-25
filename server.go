// Package wormhole
package wormhole

import (
	"context"
	"errors"
	"fmt"
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

	tunnels    sync.Map            // stores k=string;v=*tunnel
	Store      *store.Store        // api store
	DNSManager *dnsmanager.Manager // dns manager
	Issuer     *token.Issuer       // api token issuer
	Logger     logger.Logger

	cancel            context.CancelFunc
	ctx               context.Context
	tunnelHTTPRequest func(stream net.Conn, wr http.ResponseWriter, r *http.Request) error
}

func New(addr, httpAddr string) *Wormhole {
	return &Wormhole{
		addr:              addr,
		httpAddr:          httpAddr,
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

	if w.DNSManager == nil {
		return ErrNilDNSManager
	}

	if w.Store == nil {
		return ErrNilStore
	}

	if w.Logger == nil {
		w.Logger = &logger.NoopLogger{}
	}

	parentCtx, cancel := context.WithCancel(ctx)
	w.ctx = parentCtx
	w.cancel = cancel

	errch := make(chan error, 2)

	go func() {
		err := w.start()
		if err != nil && !errors.Is(err, context.Canceled) {
			w.Logger.Error("tcp server exited: " + err.Error())
			cancel()
			errch <- err
		} else {
			errch <- nil
		}
	}()

	go func() {
		err := w.StartHTTP()
		if err != nil && !errors.Is(err, context.Canceled) {
			w.Logger.Error("http server exited: " + err.Error())
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
			defer c.Close()

			if err := w.handleConn(c); err != nil {
				w.Logger.Error(fmt.Sprintf("%v\n", err))
			}
		}(conn)
	}
}

func (w *Wormhole) handleConn(conn net.Conn) error {
	w.Logger.Debug("handleConn: start")

	session, err := yamux.Server(conn, nil)
	if err != nil {
		errf := fmt.Errorf("%v: %w", ErrFailedToCreateYamuxServer, err)
		w.Logger.Error(errf.Error())
		return errf
	}
	defer session.Close()

	w.Logger.Debug("handleConn: yamux.Server created")

	stream, err := session.Accept()
	if err != nil {
		errf := fmt.Errorf("%v: %w", ErrFailedToAcceptConn, err)
		w.Logger.Error(errf.Error())
		return errf
	}

	w.Logger.Debug("handleConn: session.Accept() succeeded")

	domain, proto, ipv4, ttl, err := w.handshake(stream)
	if err != nil {
		w.Logger.Error(err.Error())
		return err
	}

	w.Logger.Debug("handleConn: handshake complete")

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

	w.Logger.Debug("handleConn: DNS record created")

	w.tunnels.Store(domain, &tunnel{proto: proto, session: session})

	var once sync.Once

	go func() {
		<-time.After(ttl)
		once.Do(func() {
			w.Logger.Debug("TTL expired, cleaning up DNS")
			err := w.DNSManager.API.DeleteDNSRecord(w.ctx, dnsRecord.ID)
			if err != nil {
				w.Logger.Error(err.Error())
			}
			session.Close()
		})
	}()

	w.Logger.Debug("handleConn: waiting for session close or context cancel")

	select {
	case <-session.CloseChan():
		w.Logger.Debug("session closed")
	case <-w.ctx.Done():
		w.Logger.Debug("context canceled")
	}

	w.Logger.Debug("handleConn: cleanup trigger reached")

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	once.Do(func() {
		w.Logger.Debug("HIT - deleting DNS record")
		err := w.DNSManager.API.DeleteDNSRecord(cleanupCtx, dnsRecord.ID)
		if err != nil {
			w.Logger.Debug("failed to delete")
		} else {
			w.Logger.Info("DNS record deleted")
		}
	})

	w.tunnels.Delete(domain)

	w.Logger.Debug("handleConn: exit")
	return nil
}
