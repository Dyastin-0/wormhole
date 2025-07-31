// Package wormhole
package wormhole

import (
	"bufio"
	"context"
	"crypto/tls"
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
	"github.com/caddyserver/certmagic"
	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog/log"
)

type Server struct {
	addr     string
	httpAddr string
	tcpAddr  string

	tunnels    sync.Map // stores k=string;v=*tunnel
	Store      *store.Store
	DNSManager *dnsmanager.Manager
	Issuer     *token.Issuer
	Logger     logger.Logger

	donech            chan bool
	cancel            context.CancelFunc
	ctx               context.Context
	tunnelHTTPRequest func(stream net.Conn, s http.ResponseWriter, r *http.Request) error
	tunnelTCPStream   func(src, dst net.Conn) error
}

func NewServer(addr, httpAddr, tcpAddr string) *Server {
	return &Server{
		addr:              addr,
		httpAddr:          httpAddr,
		tcpAddr:           tcpAddr,
		tunnelHTTPRequest: tunnelHTTPRequest,
		Logger:            &logger.NoopLogger{},
		donech:            make(chan bool, 1),
	}
}

func (s *Server) Stop() {
	s.donech <- true
}

func (s *Server) Start(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	if s.DNSManager == nil {
		return ErrNilDNSManager
	}

	if s.Store == nil {
		return ErrNilStore
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// run donech listener
	go func() {
		<-s.donech
		s.cancel()
	}()

	errch := make(chan error, 3)

	go func() {
		err := s.start()
		if err != nil && !errors.Is(err, context.Canceled) {
			s.Logger.Error("tcp server exited: " + err.Error())
			s.cancel()
		}
		errch <- err
	}()

	magic := certmagic.NewDefault()
	magic.DefaultServerName = fmt.Sprintf("*.tcp.%s", s.tcpAddr)

	go func() {
		err := s.startTCP(magic.TLSConfig())
		if err != nil {
			s.Logger.Error("tls server exited: " + err.Error())
			s.cancel()
		}
		errch <- err
	}()

	go func() {
		err := s.startHTTP()
		if err != nil && !errors.Is(err, context.Canceled) {
			s.Logger.Error("http server exited: " + err.Error())
			s.cancel()
		}
		errch <- err
	}()

	var finalErr error
	for range 3 {
		if err := <-errch; err != nil && finalErr == nil {
			finalErr = err
		}
	}

	s.Logger.Info("waiting for cleanup...")
	time.Sleep(2 * time.Second)
	s.Logger.Info("clean up done!")

	return finalErr
}

func (s *Server) start() error {
	if s.addr == "" {
		return fmt.Errorf("s.addr not set")
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("%v: %s", ErrFailedToListenToTCP, err)
	}

	go func() {
		<-s.ctx.Done()
		s.Logger.Info("context cancelled, closing tcp listener")
		listener.Close()
	}()

	s.Logger.Info("service started")

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			default:
				if errors.Is(err, net.ErrClosed) {
					return nil
				}

				s.Logger.Error(fmt.Sprintf("%s: %s", ErrFailedToAcceptConn.Error(), err))
				continue

			}
		}

		go func(c net.Conn) {
			if err := s.handleConn(c); err != nil {
				s.Logger.Error(fmt.Sprintf("%v\n", err))
			}
		}(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) error {
	session, err := yamux.Server(conn, nil)
	if err != nil {
		errf := fmt.Errorf("%v: %s", ErrFailedToCreateYamuxServer, err)
		s.Logger.Error(errf.Error())
		return errf
	}
	defer session.Close()

	stream, err := session.Accept()
	if err != nil {
		errf := fmt.Errorf("%v: %s", ErrFailedToAcceptConn, err)
		s.Logger.Error(errf.Error())
		return errf
	}

	dec := json.NewDecoder(io.LimitReader(stream, MaxJSONSize))
	enc := json.NewEncoder(stream)

	dec.DisallowUnknownFields()

	domain, proto, ipv4, ttl, err := s.handshake(enc, dec)
	if err != nil {
		s.Logger.Error(err.Error())
		return err
	}

	record := &dnsmanager.Record{
		Name:    domain,
		Content: ipv4,
		Type:    dnsmanager.RecordTypeA,
		TTL:     1,
		Proxied: false,
	}

	dnsRecord, err := s.DNSManager.API.CreateDNSRecord(s.ctx, ttl, record)
	if err != nil {
		s.Logger.Error(err.Error())
		return err
	}

	s.tunnels.Store(domain, &tunnel{proto: proto, session: session})

	ttlexpired := make(chan bool, 1)
	// handle tunnel ttl
	go func(stream net.Conn, enc *json.Encoder) {
		<-time.After(ttl)

		err = enc.Encode(&message{
			Message: MsgTunnelttlTimeout,
		})
		if err != nil {
			s.Logger.Error(fmt.Sprintf("%s: %s", ErrFailedToEncodeMessage.Error(), err.Error()))
		}
		stream.Close()

		ttlexpired <- true
	}(stream, enc)

	select {
	case <-session.CloseChan():
		s.Logger.Debug("session closed")
	case <-s.ctx.Done():
		s.Logger.Debug("context canceled")
	case <-ttlexpired:
		s.Logger.WithStr("tunnel", dnsRecord.Meta.Name).Debug("ttl expired")
	}

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = s.DNSManager.API.DeleteDNSRecord(cleanupCtx, dnsRecord.ID)
	if err != nil {
		s.Logger.Error(err.Error())
	}

	<-cleanupCtx.Done()

	s.tunnels.Delete(domain)
	return nil
}

func (s *Server) handshake(enc *json.Encoder, dec *json.Decoder) (string, string, string, time.Duration, error) {
	var msg message
	var err error

	err = dec.Decode(&msg)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("%v: %w", ErrFailedToDecodeMessage, err)
	}

	var ipv4, domain, proto string
	var ttl time.Duration

	switch msg.TunnelProto {
	case ProtoHTTP:
		domain = fmt.Sprintf("%s.%s", msg.TunnelName, s.DNSManager.API.BaseDNS())
	case ProtoTCP:
		domain = fmt.Sprintf("%s.tcp.%s", msg.TunnelName, s.DNSManager.API.BaseDNS())
	default:
		errMsg := &message{Status: StatusUnsupportedProto, Err: ErrUnsupportedProtocol.Error()}

		err = enc.Encode(errMsg)
		if err != nil {
			return "", "", "", 0, errors.Join(
				fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrUnsupportedProtocol),
				fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err),
			)
		}

		return "", "", "", 0, ErrUnsupportedProtocol
	}

	ipv4 = s.DNSManager.API.IPV4()
	ttl = 1 * time.Hour
	proto = msg.TunnelProto

	if _, exists := s.tunnels.Load(domain); exists {
		errMsg := &message{Status: StatusNameAlreadyUsed, Err: ErrTunnelNameAlreadyUsed.Error()}

		err = enc.Encode(errMsg)
		if err != nil {
			return "", "", "", 0, errors.Join(
				fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrTunnelNameAlreadyUsed),
				fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err),
			)
		}

		return "", "", "", 0, fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrTunnelNameAlreadyUsed)
	}

	err = enc.Encode(&message{
		Status:       StatusOK,
		TunnelDomain: domain,
	})
	if err != nil {
		return "", "", "", 0, fmt.Errorf("%v: %w", ErrHandshakeFailed, err)
	}

	return domain, proto, ipv4, ttl, nil
}

func (s *Server) HTTPHandler(wr http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Forwarded-Host")

	if id == "" {
		id = r.Header.Get("Host")
	}

	s.Logger.Debug("host: " + id)

	t, ok := s.tunnels.Load(id)
	if !ok {
		http.Error(wr, ErrTunnelNotFound.Error(), http.StatusNotFound)
		return
	}

	stream, err := t.(*tunnel).session.Open()
	if err != nil {
		s.Logger.Error(fmt.Sprintf("%s: %s\n", id, ErrFailedToOpenStream))
		http.Error(wr, ErrFailedToOpenStream.Error(), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	err = s.tunnelHTTPRequest(stream, wr, r)
	if err != nil {
		errf := fmt.Sprintf("tunnel error: %s", err.Error())

		s.Logger.Error(errf)
		http.Error(wr, errf, http.StatusInternalServerError)
	}
}

func (s *Server) startHTTP() error {
	if s.httpAddr == "" {
		return fmt.Errorf("s.httpAddr is not set")
	}

	server := &http.Server{
		Addr:    s.httpAddr,
		Handler: http.HandlerFunc(s.HTTPHandler),
	}

	go func() {
		<-s.ctx.Done()

		s.Logger.Info("context cancelled, shutting down http server")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			s.Logger.Error(fmt.Sprintf("%s: %v", fmt.Errorf("http server shutdown failed"), err))
		}
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("%v: %w", fmt.Errorf("http server stopped"), err)
	}

	return nil
}

func tunnelHTTPRequest(stream net.Conn, wr http.ResponseWriter, r *http.Request) error {
	err := r.Write(stream)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToWriteHTTPRequestToTunnel, err)
	}

	bufr := bufio.NewReader(stream)

	resp, err := http.ReadResponse(bufr, r)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToReadHTTPResponseFromTunnel, err)
	}

	defer resp.Body.Close()

	copyHeader(wr.Header(), resp.Header)
	io.Copy(wr, resp.Body)

	return nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (s *Server) startTCP(tlsconfig *tls.Config) error {
	if s.tcpAddr == "" {
		return fmt.Errorf("s.tcpAddr is not set")
	}

	if tlsconfig == nil {
		return ErrNilTLSConfig
	}

	listener, err := tls.Listen("tcp", s.tcpAddr, tlsconfig)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToListenToTCP, err)
	}

	go func() {
		<-s.ctx.Done()
		s.Logger.Info("context cancelled, closing tcp listener")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			default:
				if errors.Is(err, net.ErrClosed) {
					return nil
				}

				s.Logger.Error(fmt.Sprintf("%s: %s", ErrFailedToAcceptConn.Error(), err))
				continue
			}
		}

		go func() {
			err := s.TCPHandler(conn)
			if err != nil {
				s.Logger.Error(err.Error())
			}
		}()
	}
}

func (s *Server) TCPHandler(stream net.Conn) error {
	defer stream.Close()

	if s.tunnelTCPStream == nil {
		s.tunnelTCPStream = tunnelTCPStream
	}

	sni := getSNI(stream)
	if sni == "" {
		return ErrMissingSNI
	}

	t, ok := s.tunnels.Load(sni)
	if !ok {
		return ErrTunnelNotFound
	}

	dst, err := t.(*tunnel).session.Open()
	if err != nil {
		return ErrFailedToOpenStream
	}

	return s.tunnelTCPStream(stream, dst)
}

func getSNI(conn net.Conn) string {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return ""
	}

	if err := tlsConn.Handshake(); err != nil {
		log.Warn().Err(err).Msg("tls handshake failed")
		return ""
	}

	state := tlsConn.ConnectionState()
	return state.ServerName
}

func tunnelTCPStream(src, dst net.Conn) error {
	errch := make(chan error, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		errch <- stream(src, dst)
		wg.Done()
	}()

	go func() {
		errch <- stream(dst, src)
		wg.Done()
	}()

	wg.Wait()
	close(errch)

	for err := range errch {
		if err != nil {
			return err
		}
	}

	return nil
}

func stream(src, dst net.Conn) error {
	defer dst.Close()

	_, err := io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}
