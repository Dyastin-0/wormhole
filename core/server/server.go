// Package server implements the server component of the Wormhole tunneling system.
// It handles client requests to establish tunnels, manages DNS records, and forwards
// incoming connections to the appropriate client sessions using multiplexing.
package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Dyastin-0/wormhole/core/metrics"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/caddyserver/certmagic"
	"github.com/hashicorp/yamux"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/rs/zerolog/log"
)

// DefaultTunnelTTL is the default time-to-live for tunnels (1 hour).
const DefaultTunnelTTL = 1 * time.Hour

// Server manages the Wormhole tunneling service.
type Server struct {
	// addr specifies the address (:port) where the server listens for client control connections.
	addr string
	// serveAddr specifies the address (:port) where the tunneler listens for incoming tunnel traffic.
	serveAddr string
	// dnsManager manages DNS records for tunnel subdomains (e.g., creating/deleting A records).
	dnsManager dnsmanager.DNSManager
	// tunnels map domain names (e.g., "example.domain.com") to active tunnel sessions.
	tunnels *cmap.ConcurrentMap[string, *Tunnel]
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

	if s.serveAddr == "" {
		return nil, errors.New("serverAddr must be set")
	}

	if s.dnsManager == nil {
		return nil, errors.New("dnsManager must be set")
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
	magic.ManageAsync(ctx, []string{fmt.Sprintf("*.%s", s.dnsManager.BaseDomain())})

	ln, err := tls.Listen("tcp", s.serveAddr, magic.TLSConfig())
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.serveAddr, err)
	}

	go func() {
		<-ctx.Done()
		time.Sleep(5 * time.Second)
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
		time.Sleep(5 * time.Second)
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
	defer conn.Close()

	sni := getSNI(conn)
	if sni == "" {
		return fmt.Errorf("missing sni")
	}

	tunnel, ok := s.tunnels.Get(sni)
	if !ok {
		return fmt.Errorf("no tunnel for %s", sni)
	}

	return tunnel.From(ctx, conn)
}

// handleConnections accepts incoming client control connections and processes them concurrently.
func (s *Server) handleConnections(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		time.Sleep(5 * time.Second)
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
	session, err := yamux.Server(conn, nil)
	if err != nil {
		return fmt.Errorf("failed to create yamux server: %w", err)
	}
	defer session.Close()

	stream, err := session.Accept()
	if err != nil {
		return fmt.Errorf("failed to open yamux session: %w", err)
	}
	defer stream.Close()

	buf := make([]byte, proto.HeaderSize)
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	header, err := proto.DeserializeHeader(buf)
	if err != nil {
		return fmt.Errorf("failed to deserialize header: %w", err)
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
	buf := make([]byte, header.Length)
	_, err := io.ReadFull(stream, buf)
	if err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}

	req, err := proto.DeserializeRequest(buf)
	if err != nil {
		sendErr := s.sendErr(stream, "failed to deserialize request")
		if sendErr != nil {
			return errors.Join(err, sendErr)
		}
		return err
	}

	domain := fmt.Sprintf("%s.%s", req.Name, s.dnsManager.BaseDomain())

	if s.tunnels.Has(domain) {
		resp := &proto.Response{Status: proto.StatusNameTaken, Domain: domain}
		return s.sendResp(stream, resp)
	}

	record := &dnsmanager.Record{
		Type:    dnsmanager.RecordTypeA,
		Name:    req.Name,
		Proxied: false,
		TTL:     1,
	}

	dnsRecord, err := s.dnsManager.CreateDNSRecord(ctx, DefaultTunnelTTL, record)
	if err != nil {
		var sendErr error
		if errors.Is(err, dnsmanager.ErrRecordAlreadyExists) {
			resp := proto.NewResponse(proto.StatusNameTaken, uint64(DefaultTunnelTTL), domain)
			sendErr = s.sendResp(stream, resp)
			if sendErr != nil {
				return errors.Join(err, sendErr)
			}
		}
		sendErr = s.sendErr(stream, "failed to create domain")
		if sendErr != nil {
			return errors.Join(err, sendErr)
		}
		return err
	}

	tunnel := &Tunnel{
		session:   session,
		dnsRecord: dnsRecord,
		proto:     req.Proto,
		metrics:   metrics.NewMetrics(),
	}

	s.tunnels.Set(domain, tunnel)

	resp := proto.NewResponse(proto.StatusOK, uint64(DefaultTunnelTTL), domain)
	err = s.sendResp(stream, resp)
	if err != nil {
		err = fmt.Errorf("failed to send response: %w", err)
		s.tunnels.Remove(domain)
		deleteErr := s.dnsManager.DeleteDNSRecord(ctx, tunnel.dnsRecord.ID)
		if deleteErr != nil {
			return errors.Join(err, deleteErr)
		}
		return err
	}

	if header.HasFlag(proto.FlagMetrics) {
		go func() {
			er := s.streamMetrics(ctx, tunnel)
			if er != nil {
				log.Error().Err(er).Str("domain", domain).Msg("metrics stream stopped")
			}
		}()
	}

	select {
	case <-ctx.Done():
	case <-session.CloseChan():
	case <-time.After(tunnel.dnsRecord.TTL):
		log.Info().Str("domain", domain).Msg("tunnel timed out")
	}

	err = s.sendEnd(session)
	if err != nil {
		log.Error().Err(err).Msg("failed to send end")
	}

	cCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.dnsManager.DeleteDNSRecord(cCtx, tunnel.dnsRecord.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete dns record")
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

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			go s.sendMetrics(stream, tunnel)
		}
	}
}

// sendMetrics sends the latest metrics of the specified tunnel.
func (s *Server) sendMetrics(stream net.Conn, tunnel *Tunnel) error {
	metrics := proto.NewMetrics(tunnel.metrics.IngressBytes, tunnel.metrics.EgressBytes)
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
func getSNI(conn net.Conn) string {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return ""
	}

	if err := tlsConn.Handshake(); err != nil {
		state := tlsConn.ConnectionState()
		return state.ServerName
	}

	state := tlsConn.ConnectionState()
	return state.ServerName
}
