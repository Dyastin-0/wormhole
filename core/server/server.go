// Package server implements the server component of the Wormhole tunneling system.
// It handles client requests to establish tunnels, manages DNS records, and forwards
// incoming connections to the appropriate client sessions using multiplexing.
package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/wormhole/core/auth"
	"github.com/Dyastin-0/wormhole/core/metrics"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/caddyserver/certmagic"
	"github.com/hashicorp/yamux"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/rs/zerolog/log"
)

// DefaultTunnelTTL is the default time-to-live for tunnels (1 hour).
const DefaultTunnelTTL = 1 * time.Hour

var ErrNameTaken = errors.New("name taken")

var (
	headerBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, proto.HeaderSize)
			return &buf
		},
	}

	payloadBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 4096)
			return &buf
		},
	}
)

// Server manages the Wormhole tunneling service.
type Server struct {
	// addr specifies the address (:port) where the server listens for client control connections.
	addr string
	// serveAddr specifies the address (:port) where the tunneler listens for incoming tunnel traffic.
	serveAddr string
	// domain specifies the base domain name.
	domain string
	// tunnels map domain names (e.g., "example.domain.com") to active tunnel sessions.
	tunnels *cmap.ConcurrentMap[string, *Tunnel]
	// apiKeyIssuer is used to validate api key from requests.
	apiKeyIssuer *APIKeyIssuer
}

// New creates a new Server with the specified configuration options.
func New(opts ...OptFunc) (*Server, error) {
	s := &Server{}

	for _, opt := range opts {
		opt(s)
	}

	if s.tunnels == nil {
		tunnels := cmap.New[*Tunnel]()
		s.tunnels = &tunnels
	}

	if s.addr == "" {
		return nil, errors.New("addr must be set")
	}

	if s.domain == "" {
		return nil, errors.New("nil domain")
	}

	if s.serveAddr == "" {
		return nil, errors.New("serverAddr must be set")
	}

	return s, nil
}

// Run starts the server, listening for client control connections on the configured addr.
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	return s.handleConnections(ctx, ln)
}

// RunTunneler starts the tunneler, listening for incoming tunnel traffic on the configured serveAddr.
func (s *Server) RunTunneler(ctx context.Context) error {
	magic := certmagic.NewDefault()
	magic.ManageAsync(ctx, []string{fmt.Sprintf("*.%s", s.domain)})

	ln, err := tls.Listen("tcp", s.serveAddr, magic.TLSConfig())
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.serveAddr, err)
	}

	go func() {
		<-ctx.Done()
		time.Sleep(2 * time.Second)
		ln.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := ln.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return err
				}
				log.Error().Err(err).Msg("failed to accept connection")
				continue
			}
			go s.tunnel(ctx, conn)
		}
	}
}

func (s *Server) RunWithListener(ctx context.Context, ln net.Listener) error {
	return s.handleConnections(ctx, ln)
}

func (s *Server) RunTunnelerWithListener(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		time.Sleep(2 * time.Second)
		ln.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := ln.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return err
				}
				log.Error().Err(err).Msg("failed to accept connection")
				continue
			}
			go s.tunnel(ctx, conn)
		}
	}
}

// tunnel forwards an incoming connection to the appropriate client session based on the SNI.
func (s *Server) tunnel(ctx context.Context, conn net.Conn) error {
	sni, tlsConn := getSNI(conn)
	if sni == "" {
		conn.Close()
		return fmt.Errorf("missing sni")
	}
	conn = tlsConn
	defer conn.Close()

	tunnel, ok := s.tunnels.Get(sni)
	if !ok {
		return fmt.Errorf("no tunnel for %s", sni)
	}

	sniffer := &Sniff{peekN: 24}
	proto, br := sniffer.Conn(tlsConn)
	log.Debug().Str("proto", proto).Msg("detected protocol")

	if proto == ProtoHTTP && tunnel.auth.IsEnabled() {
		const maxBodySize = 10 << 20
		limited := &io.LimitedReader{R: br, N: maxBodySize}

		req, err := http.ReadRequest(bufio.NewReader(limited))
		if err != nil {
			s.sendUnauthorized(tlsConn, tunnel.auth)
			return fmt.Errorf("failed to read HTTP request: %w", err)
		}

		if !tunnel.auth.Authenticate(req) {
			s.sendUnauthorized(tlsConn, tunnel.auth)
			return fmt.Errorf("unauthorized")
		}

		var bodyBuf bytes.Buffer
		if req.ContentLength > 0 || req.Header.Get("Transfer-Encoding") == "chunked" {
			_, err = io.Copy(&bodyBuf, req.Body)
			if err != nil {
				return fmt.Errorf("failed to read request body: %w", err)
			}
			req.Body.Close()
		}

		var fullRequest bytes.Buffer

		if err := req.Write(&fullRequest); err != nil {
			return fmt.Errorf("failed to serialize request headers: %w", err)
		}

		if bodyBuf.Len() > 0 {
			_, err = io.Copy(&fullRequest, &bodyBuf)
			if err != nil {
				return fmt.Errorf("failed to copy body: %w", err)
			}
		}

		wrapped := &ConnWithReader{
			Conn: conn,
			r:    bufio.NewReader(io.MultiReader(&fullRequest, br)),
		}

		return tunnel.From(ctx, wrapped)
	}

	return tunnel.From(ctx, conn)
}

// sendUnauthorized sends the authentication challenge from the underlying authenticator.
func (s *Server) sendUnauthorized(conn net.Conn, authenticator auth.Authenticator) {
	authenticator.SendChallenge(conn)
}

// handleConnections accepts incoming client control connections and processes them concurrently.
func (s *Server) handleConnections(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		time.Sleep(2 * time.Second)
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return err
			}
			log.Error().Err(err).Msg("failed to accept connection")
			continue
		}

		go s.handleMessages(ctx, conn)
	}
}

// handleMessages processes messages from a client connection using a yamux session.
func (s *Server) handleMessages(ctx context.Context, conn net.Conn) error {
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.KeepAliveInterval = 1 * time.Second

	session, err := yamux.Server(conn, yamuxConfig)
	if err != nil {
		return fmt.Errorf("failed to create yamux server: %w", err)
	}
	defer session.Close()

	stream, err := session.Accept()
	if err != nil {
		return fmt.Errorf("failed to open yamux session: %w", err)
	}
	defer stream.Close()

	bufPtr := headerBufferPool.Get().(*[]byte)
	defer headerBufferPool.Put(bufPtr)

	_, err = io.ReadFull(stream, *bufPtr)
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	header, err := proto.DeserializeHeader(*bufPtr)
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	switch header.Type {
	case proto.TypeRequest:
		return s.handleRequest(ctx, stream, session, header)
	default:
		return fmt.Errorf("unexpected header type: %v", header.Type)
	}
}

// handleRequest processes a client’s tunnel request.
func (s *Server) handleRequest(ctx context.Context, stream net.Conn, session *yamux.Session, header *proto.Header) error {
	bufPtr := payloadBufferPool.Get().(*[]byte)
	defer payloadBufferPool.Put(bufPtr)

	*bufPtr = (*bufPtr)[:header.Length]

	_, err := io.ReadFull(stream, *bufPtr)
	if err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}

	req, err := proto.DeserializeRequest(*bufPtr)
	if err != nil {
		sendErr := s.sendErr(stream, fmt.Sprintf("failed to deserialize request: %s", err.Error()))
		if sendErr != nil {
			return errors.Join(err, sendErr)
		}
		return err
	}

	domain := fmt.Sprintf("%s.%s", req.Name, s.domain)

	if s.tunnels.Has(domain) {
		resp := &proto.Response{Status: proto.StatusNameTaken, Domain: domain}
		sendErr := s.sendResp(stream, resp)
		if sendErr != nil {
			return sendErr
		}

		return ErrNameTaken
	}

	ttl := DefaultTunnelTTL

	if req.APIKey != "" {
		claims, errr := s.apiKeyIssuer.Validate(req.APIKey)
		if errr == nil {
			// if ttl from claims is zero (0), it is a priviledged api key.
			// we will use the ttl from the client request if provided.
			if claims.TTL == 0 {
				if req.TTL > 0 {
					ttl = time.Duration(req.TTL) * time.Hour
					log.Info().Str("domain", domain).Uint64("ttl_hours", req.TTL).Msg("using client-specified ttl")
				} else {
					log.Debug().Str("domain", domain).Msg("privileged key with no ttl specified")
				}
			} else {
				ttl = time.Duration(claims.TTL) * time.Hour
			}
		} else {
			log.Error().Err(err).Str("domain", domain).Msg("api key validation")
			sendErr := s.sendErr(stream, errr.Error())
			if sendErr != nil {
				return errors.Join(err, sendErr)
			}
			return err
		}
	}

	var authenticator auth.Authenticator

	switch req.AuthType {
	case proto.AuthTypeBasic:
		authenticator, err = auth.NewBasicAuth(req.AuthUsername, req.AuthPassword)
		if err != nil {
			authenticator = &auth.NoAuth{}
		}
	case proto.AuthTypeBearer:
		authenticator, err = auth.NewBearerAuth(req.AuthToken)
		if err != nil {
			authenticator = &auth.NoAuth{}
		}
	case proto.AuthTypeNone:
		fallthrough
	default:
		authenticator = &auth.NoAuth{}
	}

	tunnel := &Tunnel{
		session: session,
		proto:   req.Proto,
		ttl:     ttl,
		auth:    authenticator,
		metrics: metrics.New(),
	}

	s.tunnels.Set(domain, tunnel)

	resp := proto.NewResponse(proto.StatusOK, uint64(tunnel.ttl), domain)
	err = s.sendResp(stream, resp)
	if err != nil {
		err = fmt.Errorf("failed to send response: %w", err)
		s.tunnels.Remove(domain)
		return err
	}

	if header.HasFlag(proto.FlagMetrics) {
		metricsCtx, metricsCancel := context.WithCancel(ctx)
		defer metricsCancel()

		go func(ctx context.Context, tunnel *Tunnel) {
			er := s.streamMetrics(ctx, tunnel)
			if er != nil {
				log.Error().Err(er).Str("domain", domain).Msg("metrics stream stopped")
			}
		}(metricsCtx, tunnel)
	}

	select {
	case <-ctx.Done():
	case <-session.CloseChan():
		log.Info().Str("domain", domain).Msg("session closed")
	case <-time.After(tunnel.ttl):
		log.Info().Str("domain", domain).Msg("tunnel timed out")
	}

	err = s.sendEnd(session)
	if err != nil {
		log.Error().Err(err).Msg("failed to send end")
	}

	s.tunnels.Remove(domain)

	return nil
}

func (s *Server) sendEnd(session *yamux.Session) error {
	stream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open yamux stream: %w", err)
	}
	defer stream.Close()

	header := proto.NewHeader(proto.TypeEnd, 0)
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize header: %w", err)
	}

	_, err = stream.Write(serializedHeader)
	if err != nil {
		return fmt.Errorf("failed to write end: %w", err)
	}

	return nil
}

// sendResp sends a response with a TypeResponse header and the provided Response payload.
func (s *Server) sendResp(stream net.Conn, resp *proto.Response) error {
	serializedResponse, err := proto.SerializeResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to serialize response: %w", err)
	}

	header := proto.NewHeader(proto.TypeResponse, uint64(len(serializedResponse)))
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize header: %w", err)
	}

	data := append(serializedHeader, serializedResponse...)

	_, err = stream.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// sendErr sends an error response with a TypeError header and the provided message.
func (s *Server) sendErr(stream net.Conn, message string) error {
	defer stream.Close()
	stream.SetDeadline(time.Now().Add(5 * time.Second))

	header := proto.NewHeader(proto.TypeError, uint64(len(message)))
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize header: %w", err)
	}

	data := append(serializedHeader, []byte(message)...)

	_, err = stream.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write error response: %w", err)
	}

	return nil
}

// streamMetrics streams the tunnel metrics to the tunnel on interval.
func (s *Server) streamMetrics(ctx context.Context, tunnel *Tunnel) error {
	stream, err := tunnel.session.Open()
	if err != nil {
		return fmt.Errorf("failed to open yamux stream: %w", err)
	}
	defer stream.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.sendMetrics(stream, tunnel); err != nil {
				return fmt.Errorf("failed to send metrics: %w", err)
			}
		}
	}
}

// sendMetrics sends the latest metrics of the specified tunnel.
func (s *Server) sendMetrics(stream net.Conn, tunnel *Tunnel) error {
	metrics := proto.NewMetrics(
		tunnel.metrics.GetIngressBytes(),
		tunnel.metrics.GetEgressBytes(),
		uint64(tunnel.metrics.GetUptime()),
		tunnel.metrics.GetConnectionCount(),
		uint32(tunnel.metrics.GetActiveConnections()),
	)
	serializedMetrics, err := proto.SerializeMetrics(metrics)
	if err != nil {
		return fmt.Errorf("failed to serialize metrics: %w", err)
	}

	header := proto.NewHeader(proto.TypeMetrics, uint64(len(serializedMetrics)))
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize header: %w", err)
	}

	_, err = stream.Write(serializedHeader)
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	_, err = stream.Write(serializedMetrics)
	if err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}

	return nil
}

// getSNI extracts the Server Name Indication (SNI) from a TLS connection.
func getSNI(conn net.Conn) (string, *tls.Conn) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return "", nil
	}

	if err := tlsConn.Handshake(); err != nil {
		state := tlsConn.ConnectionState()
		return state.ServerName, tlsConn
	}

	state := tlsConn.ConnectionState()

	return state.ServerName, tlsConn
}
