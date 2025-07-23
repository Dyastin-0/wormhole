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

	"github.com/Dyastin-0/wormhole/api/db"
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
	defer conn.Close()

	session, err := yamux.Server(conn, nil)
	if err != nil {
		errf := fmt.Errorf("%v: %w", ErrFailedToCreateYamuxServer, err)
		w.Logger.Error(errf.Error())
		return errf
	}

	stream, err := session.Accept()
	if err != nil {
		errf := fmt.Errorf("%v: %w", ErrFailedToAcceptConn, err)
		w.Logger.Error(errf.Error())
		return errf
	}

	msg, payload, err := w.handshake(stream)
	if err != nil {
		w.Logger.Error(err.Error())
		return err
	}

	var domain, ipv4 string
	var ttl time.Duration

	if payload != nil && msg.TunnelID != "" {
		userID := (*payload)[token.PayloadID].(string)

		param := &db.GetTunnelParams{
			ID:     msg.TunnelID,
			UserID: userID,
		}

		res, err := w.Store.Tunnel.Get(w.ctx, param)
		if err != nil {
			w.Logger.Error(err.Error())
			return err
		}

		domain = res.Domain
		ipv4 = res.Ipv4
		ttl = 24 * time.Hour
	} else {
		domain = fmt.Sprintf("%s.%s", msg.TunnelName, w.DNSManager.API.BaseDNS())
		ipv4 = w.DNSManager.API.IPV4()
		ttl = 1 * time.Hour
	}

	record := &dnsmanager.Record{
		Name:    domain,
		Content: ipv4,
		Type:    dnsmanager.RecordTypeA,
		TTL:     720,
		Proxied: false,
	}

	dnsRecord, err := w.DNSManager.API.CreateDNSRecord(w.ctx, ttl, record)
	if err != nil {
		w.Logger.Error(err.Error())
		return err
	}

	w.tunnels.Store(domain, &tunnel{proto: msg.TunnelProto, session: session})

	var once sync.Once

	// start a ttl watcher, if ttl is done delete record and close the session
	go func() {
		select {
		case <-time.After(ttl):
			once.Do(func() {
				err := w.DNSManager.API.DeleteDNSRecord(w.ctx, dnsRecord.ID)
				if err != nil {
					w.Logger.Error(err.Error())
				}
				session.Close()
			})
		}
	}()

	<-session.CloseChan()

	// forcibly delete dns if session is closed
	once.Do(func() {
		err := w.DNSManager.API.DeleteDNSRecord(w.ctx, dnsRecord.ID)
		if err != nil {
			w.Logger.Error(err.Error())
		}
		session.Close()
	})

	w.tunnels.Delete(domain)

	return nil
}
