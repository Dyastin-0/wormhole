// Package server implements the server component of the Wormhole tunneling system.
// It handles client requests to establish tunnels, and forwards
// incoming connections to the appropriate client sessions using multiplexing.
package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Dyastin-0/wormhole/core/auth"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/metrics"
	"github.com/Dyastin-0/wormhole/observer"
	"github.com/Dyastin-0/wormhole/stream"
	"github.com/hashicorp/yamux"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// DefaultTunnelTTL is the default time-to-live for tunnels (1 hour).
const (
	DefaultTunnelTTL = 1 * time.Hour
	// Used for allocating TCP listeners for TCP tunnels.
	MinPort      = 30000
	MaxPort      = 31000
	MaxRandRetry = 50
)

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
	// observer is used for telemetry.
	observer observer.Observer
	// tracer is the OpenTelemetry tracer for distributed tracing.
	tracer trace.Tracer
	// allowTCP specifies if this server should accept plain TCP tunnel requests.
	allowTCP bool
	// portAllocator handles port allocation for plain TCP tunnels.
	portAllocator *PortAllocator
	// tlsConfig is used to terminate tunnel connections.
	tlsConfig *tls.Config
}

// New creates a new Server with the specified configuration options.
func New(opts ...OptFunc) (*Server, error) {
	s := &Server{
		observer: &observer.NoopObserver{},
		tracer:   noop.NewTracerProvider().Tracer("wormhole-server"),
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.allowTCP {
		// Use defaults for now
		s.portAllocator = NewPortAllocator(MinPort, MaxPort, MaxRandRetry)
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
		return nil, errors.New("serveAddr must be set")
	}

	return s, nil
}

// Run starts the server, listening for client control connections on the configured addr.
func (s *Server) Run(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "server.Run",
		trace.WithAttributes(
			attribute.String("addr", s.addr),
			attribute.String("domain", s.domain),
		),
	)
	defer span.End()

	// NOTE: this listens for plain TCP connections
	// and should be put behind a reverse proxy.
	// all traffic for <domain> should be routed here.
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to listen")
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	span.SetStatus(codes.Ok, "listening for connections")
	return s.handleConnections(ctx, ln)
}

// RunTunneler starts the tunneler, listening for incoming tunnel traffic on the configured serveAddr.
func (s *Server) RunTunneler(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "server.RunTunneler",
		trace.WithAttributes(
			attribute.String("serve_addr", s.serveAddr),
			attribute.String("domain", s.domain),
		),
	)
	defer span.End()

	// NOTE: this listens for TLS connections.
	// all traffic for *.<domain> should be routed here.
	// Additionally, CNAMES that is registered by clients via the `--url` and '--tls-passthrough` flags
	// should be handled. if this listener is behind a reverse proxy (you'll need a tcp/tls proxy)
	// you need to somehow route that CNAME here. a clever hack is to
	// route domains that is not registered on your reverse proxy to a default.
	ln, err := net.Listen("tcp", s.serveAddr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to listen")
		return fmt.Errorf("failed to listen on %s: %w", s.serveAddr, err)
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	span.SetStatus(codes.Ok, "listening for tunnel traffic")

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

// RunObserver starts the metrics/health HTTP server.
func (s *Server) RunObserver(ctx context.Context, addr string) error {
	ctx, span := s.tracer.Start(ctx, "server.RunObserver",
		trace.WithAttributes(attribute.String("addr", addr)),
	)
	defer span.End()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	observerServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func(ctx context.Context) {
		<-ctx.Done()

		// This may be adjusted based on how often observer scrapes metrics,
		// you could miss metrics when the server closes faster than
		// how often observer (e.g., prometheus) scrape metrics.
		time.Sleep(2 * time.Second)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := observerServer.Shutdown(shutdownCtx); err != nil {
			log.Err(err).Msg("observer shutdown")
		}
	}(ctx)

	span.SetStatus(codes.Ok, "observer server started")
	return observerServer.ListenAndServe()
}

// tunnel forwards an incoming connection to the appropriate client session based on the SNI.
func (s *Server) tunnel(ctx context.Context, conn net.Conn) error {
	defer conn.Close()

	ctx, span := s.tracer.Start(ctx, "server.tunnel",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	start := time.Now()

	conn, err := stream.TLS(conn)
	if err != nil {
		span.SetStatus(codes.Error, "failed to read client hello message")
		return err
	}

	sni := conn.(*stream.TLSConn).Host()
	if sni == "" {
		err = fmt.Errorf("missing sni")
		span.RecordError(err)
		span.SetStatus(codes.Error, "missing sni")
		return err
	}
	span.SetAttributes(attribute.String("sni", sni))

	tunnel, ok := s.tunnels.Get(sni)
	if !ok {
		span.RecordError(err)
		span.SetStatus(codes.Error, "tunnel not found")

		// if tunnel is not found, we immediately terminate it
		// and see if we can send an HTTP response.
		conn = tls.Server(conn, s.tlsConfig)

		var detectedProtocol string
		detectedProtocol, conn = stream.Conn(conn)

		if detectedProtocol == stream.ProtoHTTP {
			s.writeNoTunnel(conn, sni)
		}

		return nil
	}

	if tunnel.metrics != nil {
		tunnel.metrics.IncrementConnections()
		defer tunnel.metrics.DecrementActiveConnections()
	}

	allowTLSPassthrough := tunnel.allowTLSPassthrough
	span.SetAttributes(attribute.Bool("allow_tls_passthrough", allowTLSPassthrough))

	if !allowTLSPassthrough {
		// If TLS passthrough is disabled, we terminate TLS here
		// using our own TLS config.
		conn = tls.Server(conn, s.tlsConfig)
	}

	var detectedProtocol string
	detectedProtocol, conn = stream.Conn(conn)

	protoStr := proto.ProtoString(tunnel.proto)
	span.SetAttributes(
		attribute.String("protocol", protoStr),
		attribute.String("domain", tunnel.domain),
		attribute.Bool("allow_http", tunnel.allowHTTP),
		attribute.Bool("has_auth", tunnel.auth != nil),
	)

	s.observer.RecordConnectionStart(tunnel.domain, protoStr)
	defer func() {
		s.observer.RecordConnectionEnd(sni, protoStr, time.Since(start))
	}()

	allowHTTP := tunnel.allowHTTP || tunnel.proto == proto.ProtoHTTP
	isHTTP := detectedProtocol == stream.ProtoHTTP
	span.SetAttributes(attribute.Bool("is_http", isHTTP))

	if !allowTLSPassthrough && isHTTP && !allowHTTP {
		err = fmt.Errorf("http not allowed on tcp tunnel")
		span.RecordError(err)
		span.SetStatus(codes.Error, "http forbidden on tcp tunnel")
		s.writeForbidden(conn, sni)
		return err
	}

	if !allowTLSPassthrough && isHTTP && tunnel.auth != nil {
		err = s.httpAuthProxy(ctx, conn, tunnel)
		return err
	}

	if !allowTLSPassthrough && isHTTP && tunnel.httpLogch != nil {
		err = tunnel.ProxyWithInspect(ctx, conn)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "proxy with inspect failed")
		} else {
			span.SetStatus(codes.Ok, "completed")
		}
		return err
	}

	err = tunnel.Proxy(ctx, conn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "proxy failed")
	} else {
		span.SetStatus(codes.Ok, "completed")
	}

	return err
}

func (s *Server) httpAuthProxy(ctx context.Context, conn net.Conn, tunnel *Tunnel) error {
	defer conn.Close()

	ctx, span := s.tracer.Start(ctx, "server.httpAuthProxy",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	httpConn, err := stream.HTTP(conn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read http request")
		s.sendUnauthorized(conn, tunnel.auth)
		return fmt.Errorf("failed to read http request: %w", err)
	}

	req := httpConn.Request

	span.SetAttributes(
		attribute.String("http.method", req.Method),
		attribute.String("http.path", req.URL.Path),
	)

	if !tunnel.auth.Authenticate(req) {
		err = fmt.Errorf("unauthorized")
		span.RecordError(err)
		span.SetStatus(codes.Error, "authentication failed")
		span.SetAttributes(attribute.Int("http.status_code", http.StatusUnauthorized))
		s.sendUnauthorized(conn, tunnel.auth)
		return err
	}

	span.SetAttributes(attribute.Bool("authenticated", true))

	var fullRequest bytes.Buffer
	if err = req.Write(&fullRequest); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize request")
		return fmt.Errorf("failed to serialize request: %w", err)
	}

	if tunnel.httpLogch != nil {
		err = tunnel.ProxyWithInspect(ctx, httpConn)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "proxy with inspect failed")
		} else {
			span.SetStatus(codes.Ok, "completed")
		}
		return err
	}

	err = tunnel.Proxy(ctx, httpConn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "proxy failed")
	} else {
		span.SetStatus(codes.Ok, "completed")
	}
	return err
}

// streamHTTPLogs streams HTTP request logs to the client.
func (s *Server) streamHTTPLogs(ctx context.Context, tunnel *Tunnel) error {
	ctx, span := s.tracer.Start(ctx, "server.streamHTTPLogs",
		trace.WithAttributes(attribute.String("domain", tunnel.domain)),
	)
	defer span.End()

	stream, err := tunnel.session.Open()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to open yamux stream")
		return fmt.Errorf("failed to open yamux stream: %w", err)
	}
	defer stream.Close()

	span.SetStatus(codes.Ok, "streaming http logs")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tunnel.session.CloseChan():
			return nil
		case httpLog := <-tunnel.httpLogch:
			s.observer.RecordHTTPRequest(tunnel.domain, httpLog.Method, fmt.Sprint(httpLog.Status), time.Duration(httpLog.Duration))
			if err := s.sendHTTPLog(stream, httpLog); err != nil {
				span.RecordError(err)
				return fmt.Errorf("failed to send http log: %w", err)
			}
		}
	}
}

// sendHTTPLog sends an HTTP log entry to the client.
func (s *Server) sendHTTPLog(stream net.Conn, httpLog *HTTPLog) error {
	serialized, err := proto.SerializeHTTPLog(httpLog.HTTPLog)
	if err != nil {
		return fmt.Errorf("failed to serialize http log: %w", err)
	}

	header := proto.NewHeader(proto.TypeHTTPLog, uint64(len(serialized)))
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize header: %w", err)
	}

	_, err = stream.Write(serializedHeader)
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	_, err = stream.Write(serialized)
	if err != nil {
		return fmt.Errorf("failed to write http log: %w", err)
	}

	return nil
}

// sendUnauthorized sends the authentication challenge from the underlying authenticator.
func (s *Server) sendUnauthorized(conn net.Conn, authenticator auth.Authenticator) {
	authenticator.SendChallenge(conn)
}

// handleConnections accepts incoming client control connections and processes them concurrently.
func (s *Server) handleConnections(ctx context.Context, ln net.Listener) error {
	ctx, span := s.tracer.Start(ctx, "server.handleConnections")
	defer span.End()

	go func() {
		<-ctx.Done()
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
	defer conn.Close()

	ctx, span := s.tracer.Start(ctx, "server.handleMessages",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	var detectedProtocol string
	detectedProtocol, conn = stream.Conn(conn)
	span.SetAttributes(attribute.String("detected_protocol", string(detectedProtocol)))

	if detectedProtocol == stream.ProtoHTTP {
		s.writeHomePage(conn)
		span.SetStatus(codes.Ok, "served homepage")
		return nil
	}

	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = false

	session, err := yamux.Server(conn, yamuxConfig)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create yamux server")
		return fmt.Errorf("failed to create yamux server: %w", err)
	}
	defer session.Close()

	stream, err := session.Accept()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to accept yamux stream")
		return fmt.Errorf("failed to open yamux session: %w", err)
	}
	defer stream.Close()

	bufPtr := headerBufferPool.Get().(*[]byte)
	defer headerBufferPool.Put(bufPtr)

	_, err = io.ReadFull(stream, *bufPtr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read header")
		return fmt.Errorf("failed to read header: %w", err)
	}

	header, err := proto.DeserializeHeader(*bufPtr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deserialize header")
		return fmt.Errorf("failed to read header: %w", err)
	}

	span.SetAttributes(attribute.String("header_type", fmt.Sprintf("%v", header.Type)))

	switch header.Type {
	case proto.TypeRequest:
		return s.handleRequest(ctx, stream, session, header)
	default:
		err := fmt.Errorf("unexpected header type: %v", header.Type)
		span.RecordError(err)
		span.SetStatus(codes.Error, "unexpected header type")
		return err
	}
}

// handleRequest processes a client's tunnel request.
func (s *Server) handleRequest(ctx context.Context, stream net.Conn, session *yamux.Session, header *proto.Header) error {
	ctx, span := s.tracer.Start(ctx, "server.handleRequest")
	defer span.End()

	bufPtr := payloadBufferPool.Get().(*[]byte)
	defer payloadBufferPool.Put(bufPtr)

	*bufPtr = (*bufPtr)[:header.Length]

	_, err := io.ReadFull(stream, *bufPtr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read request")
		return fmt.Errorf("failed to read request: %w", err)
	}

	req, err := proto.DeserializeRequest(*bufPtr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deserialize request")
		sendErr := s.sendErr(stream, fmt.Sprintf("failed to deserialize request: %s", err.Error()))
		if sendErr != nil {
			return errors.Join(err, sendErr)
		}
		return err
	}

	domain := fmt.Sprintf("%s.%s", req.Name, s.domain)
	span.SetAttributes(
		attribute.String("domain", domain),
		attribute.String("requested_name", req.Name),
		attribute.String("url", req.URL),
		attribute.String("protocol", proto.ProtoString(req.Proto)),
	)

	if header.HasFlag(proto.FlagTLSPassthrough) {
		var u *url.URL
		u, err = url.ParseRequestURI(req.URL)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "invalid url")

			resp := &proto.Response{Status: proto.StatusInvalidURL, Domain: domain}
			sendErr := s.sendResp(stream, resp)
			if sendErr != nil {
				return errors.Join(err, sendErr)
			}
			return err
		}

		req.URL = u.Host
	}

	isTCP := req.Proto == proto.ProtoTCP

	if !isTCP && s.tunnels.Has(domain) {
		err = errors.New("name taken")
		span.RecordError(err)
		span.SetStatus(codes.Error, "name already taken")

		resp := &proto.Response{Status: proto.StatusNameTaken, Domain: domain}
		sendErr := s.sendResp(stream, resp)
		if sendErr != nil {
			return errors.Join(err, sendErr)
		}

		return err
	}

	ttl := DefaultTunnelTTL

	if req.APIKey != "" {
		span.SetAttributes(attribute.Bool("has_api_key", true))

		claims, errr := s.apiKeyIssuer.Validate(req.APIKey)
		if errr != nil {
			span.RecordError(errr)
			span.SetStatus(codes.Error, "api key validation failed")
			log.Error().Err(err).Str("domain", domain).Msg("api key validation")
			sendErr := s.sendErr(stream, errr.Error())
			if sendErr != nil {
				return errors.Join(err, sendErr)
			}
			return err
		}

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
	}

	span.SetAttributes(attribute.Int64("ttl_seconds", int64(ttl.Seconds())))

	var authenticator auth.Authenticator

	switch req.AuthType {
	case proto.AuthTypeBasic:
		span.SetAttributes(attribute.String("auth_type", "basic"))
		authenticator, err = auth.NewBasicAuth(req.AuthUsername, req.AuthPassword)
		if err != nil {
			log.Warn().Err(err).Msg("failed to use basic auth")
		}
	case proto.AuthTypeBearer:
		span.SetAttributes(attribute.String("auth_type", "bearer"))
		authenticator, err = auth.NewBearerAuth(req.AuthToken)
		if err != nil {
			log.Warn().Err(err).Msg("failed to use bearer auth")
		}
	case proto.AuthTypeNone:
		span.SetAttributes(attribute.String("auth_type", "none"))
		fallthrough
	default:
		log.Warn().Uint8("authType", req.AuthType).Msg("unexpected auth type")
		authenticator = nil
	}

	tunnel := &Tunnel{
		session:             session,
		controlStream:       stream,
		proto:               req.Proto,
		ttl:                 ttl,
		auth:                authenticator,
		domain:              domain,
		createdAt:           time.Now(),
		allowHTTP:           header.HasFlag(proto.FlagAllowHTTP),
		allowTLSPassthrough: header.HasFlag(proto.FlagTLSPassthrough),
	}

	span.SetAttributes(attribute.Bool("allow_http", tunnel.allowHTTP))
	span.SetAttributes(attribute.Bool("tls_passthrough", tunnel.allowTLSPassthrough))

	if header.HasFlag(proto.FlagHTTPLog) {
		tunnel.httpLogch = make(chan *HTTPLog, 100)
		span.SetAttributes(attribute.Bool("http_log_enabled", true))

		go func(ctx context.Context, tunnel *Tunnel) {
			er := s.streamHTTPLogs(ctx, tunnel)
			if er != nil {
				log.Error().Err(er).Str("domain", domain).Msg("http log stream stopped")
			}
		}(ctx, tunnel)
	}

	if header.HasFlag(proto.FlagMetrics) {
		tunnel.metrics = metrics.New()
		span.SetAttributes(attribute.Bool("metrics_enabled", true))

		go func(ctx context.Context, tunnel *Tunnel) {
			er := s.streamMetrics(ctx, tunnel)
			if er != nil {
				log.Error().Err(er).Str("domain", domain).Msg("metrics stream stopped")
			}
		}(ctx, tunnel)

		go func(ctx context.Context, tunnel *Tunnel) {
			er := s.handlePingStream(ctx, tunnel)
			if er != nil {
				log.Error().Err(er).Str("domain", domain).Msg("ping stream stopped")
			}
		}(ctx, tunnel)
	}

	if isTCP {
		span.SetAttributes(attribute.Bool("has_tcp_flag", true))

		port, listener, errr := s.portAllocator.AllocateListener()
		if errr != nil {
			span.RecordError(errr)
			span.SetStatus(codes.Error, "failed to allocate port")
			s.sendErr(stream, fmt.Sprintf("failed to allocate listener: %s", errr.Error()))
			return errr
		}

		tunnel.port = port
		tunnel.tcpListener = listener

		span.SetAttributes(attribute.Int("tcp_port", port))

		// With plain TCP we don't necessarily need to mark domain as used
		// since we allocate a dedicated port for it, any subdomain will
		// resolve to the same IP anyway.
		go s.handleTCPTunnel(ctx, tunnel)
	} else {
		// For tunnels with TLS passthrough, we only need to
		// use the given URL, as they will be handling TLS termination.
		if header.HasFlag(proto.FlagTLSPassthrough) {
			s.tunnels.SetIfAbsent(req.URL, tunnel)
		} else {
			s.tunnels.SetIfAbsent(domain, tunnel)
		}
	}

	protoStr := proto.ProtoString(req.Proto)
	s.observer.RecordTunnelCreated(protoStr)

	resp := proto.NewResponse(proto.StatusOK, uint64(tunnel.ttl), domain)
	if isTCP && tunnel.port > 0 {
		resp.Port = uint16(tunnel.port)
	} else {
		// set to standard TLS port,
		// assuming s.serveAddr is behind a reverse proxy.
		// this does not affect any logical feature,
		// and is only displayed on the client side.
		resp.Port = uint16(443)
	}

	err = s.sendResp(stream, resp)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send response")
		err = fmt.Errorf("failed to send response: %w", err)
		if isTCP && tunnel.tcpListener != nil {
			tunnel.tcpListener.Close()
		} else {
			s.tunnels.Remove(domain)
		}
		s.observer.RecordTunnelClosed(protoStr, "error", 0)
		return err
	}

	span.SetStatus(codes.Ok, "tunnel established")

	var closeReason string
	select {
	case <-ctx.Done():
		closeReason = "context_cancelled"
	case <-session.CloseChan():
		closeReason = "client_disconnect"
		log.Info().Str("domain", domain).Msg("session closed")
	case <-time.After(tunnel.ttl):
		closeReason = "timeout"
		log.Info().Str("domain", domain).Msg("tunnel timed out")
	}

	span.SetAttributes(attribute.String("close_reason", closeReason))

	err = s.sendEnd(tunnel)
	if err != nil {
		log.Error().Err(err).Msg("failed to send end")
	}

	if isTCP {
		tunnel.tcpListener.Close()
	} else {
		if header.HasFlag(proto.FlagTLSPassthrough) {
			s.tunnels.Remove(req.URL)
		} else {
			s.tunnels.Remove(domain)
		}
	}

	duration := time.Since(tunnel.createdAt)
	span.SetAttributes(attribute.Int64("tunnel_duration_seconds", int64(duration.Seconds())))
	s.observer.RecordTunnelClosed(protoStr, closeReason, duration)

	return nil
}

// handleTCPTunnel listens for connections from the allocated listener.
func (s *Server) handleTCPTunnel(ctx context.Context, tunnel *Tunnel) {
	ctx, span := s.tracer.Start(ctx, "server.handleTCPTunnel",
		trace.WithAttributes(
			attribute.String("domain", tunnel.domain),
			attribute.Int("tcp_port", tunnel.port),
		),
	)
	defer span.End()

	log.Info().
		Str("domain", tunnel.domain).
		Int("port", tunnel.port).
		Msg("TCP tunnel listening")

	for {
		select {
		case <-ctx.Done():
			return
		case <-tunnel.session.CloseChan():
			return
		default:
			conn, err := tunnel.tcpListener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error().Err(err).Msg("failed to accept TCP tunnel connection")
				continue
			}
			go s.tunnelTCP(ctx, conn, tunnel)
		}
	}
}

func (s *Server) tunnelTCP(ctx context.Context, conn net.Conn, tunnel *Tunnel) {
	defer conn.Close()
	ctx, span := s.tracer.Start(ctx, "server.tunnelTCP",

		trace.WithAttributes(attribute.String("domain", tunnel.domain)),
	)
	defer span.End()

	if tunnel.metrics != nil {
		tunnel.metrics.IncrementConnections()
		defer tunnel.metrics.DecrementActiveConnections()
	}

	var proto string
	proto, conn = stream.Conn(conn)
	if tunnel.allowHTTP && tunnel.auth != nil && proto == stream.ProtoHTTP {
		s.httpAuthProxy(ctx, conn, tunnel)
		return
	}

	if tunnel.allowHTTP && proto == stream.ProtoHTTP {
		err := tunnel.ProxyWithInspect(ctx, conn)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "proxy with inspect failed")
		} else {
			span.SetStatus(codes.Ok, "completed")
		}
		return
	}

	err := tunnel.Proxy(ctx, conn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "proxy failed")
	} else {
		span.SetStatus(codes.Ok, "completed")
	}
}

func (s *Server) sendEnd(tunnel *Tunnel) error {
	stream := tunnel.controlStream

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
	ctx, span := s.tracer.Start(ctx, "server.streamMetrics",
		trace.WithAttributes(attribute.String("domain", tunnel.domain)),
	)
	defer span.End()

	stream, err := tunnel.session.Open()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to open yamux stream")
		return fmt.Errorf("failed to open yamux stream: %w", err)
	}
	defer stream.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	span.SetStatus(codes.Ok, "streaming metrics")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ingressDelta := tunnel.metrics.GetIngressBytesDelta()
			egressDelta := tunnel.metrics.GetEgressBytesDelta()

			if ingressDelta > 0 || egressDelta > 0 {
				s.observer.RecordTraffic(tunnel.domain, ingressDelta, egressDelta)
			}

			if err := s.sendMetrics(stream, tunnel); err != nil {
				span.RecordError(err)
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
		tunnel.metrics.GetRTT(),
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

// handlePingStream accepts pings from client and measures RTT.
func (s *Server) handlePingStream(ctx context.Context, tunnel *Tunnel) error {
	ctx, span := s.tracer.Start(ctx, "server.handlePingStream",
		trace.WithAttributes(attribute.String("domain", tunnel.domain)),
	)
	defer span.End()

	stream, err := tunnel.session.OpenStream()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to open ping stream")
		return fmt.Errorf("failed to open ping stream: %w", err)
	}
	defer stream.Close()

	log.Debug().Msg("ping stream established")
	span.SetStatus(codes.Ok, "ping stream established")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	bufPtr := headerBufferPool.Get().(*[]byte)
	defer headerBufferPool.Put(bufPtr)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tunnel.session.CloseChan():
			return nil
		case <-ticker.C:
			stream.SetDeadline(time.Now().Add(1 * time.Second))

			pingHeader := proto.NewHeader(proto.TypePing, 0)
			serialized, err := proto.SerializeHeader(pingHeader)
			if err != nil {
				span.RecordError(err)
				log.Error().Err(err).Msg("failed to serialize ping")
				continue
			}

			start := time.Now()
			_, err = stream.Write(serialized)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to write ping")
				log.Error().Err(err).Msg("failed to write ping")
				tunnel.session.Close()
				return err
			}

			_, err = io.ReadFull(stream, *bufPtr)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to read pong")
				log.Error().Err(err).Msg("failed to read pong")
				tunnel.session.Close()
				return err
			}

			rtt := time.Since(start)

			header, err := proto.DeserializeHeader(*bufPtr)
			if err != nil {
				span.RecordError(err)
				log.Error().Err(err).Msg("failed to deserialize pong")
				continue
			}

			if header.Type != proto.TypePong {
				log.Warn().Uint8("type", header.Type).Msg("unexpected message type, expected pong")
				continue
			}

			tunnel.metrics.SetRTT(uint32(rtt.Microseconds()))
			s.observer.UpdateRTT(tunnel.domain, uint32(rtt.Microseconds()))

			span.SetAttributes(attribute.Int64("rtt_microseconds", rtt.Microseconds()))
		}
	}
}
