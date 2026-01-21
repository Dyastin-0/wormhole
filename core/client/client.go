// Package client implements the client component of the Wormhole tunneling system.
// It establishes tunnel to a server and forwards incoming multiplexed connections
// to a target address.
package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/proxy"
	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog/log"
)

var (
	ErrNameTaken        = errors.New("name taken")
	ErrUnsupportedProto = errors.New("unsupported protocol")
)

// Client represents a Wormhole client that establishes and manages a tunnel.
type Client struct {
	// addr is the TCP address (host:port) of the Wormhole server.
	addr string
	// targetAddr is the TCP address (host:port) where incoming connections are forwarded.
	targetAddr string
	// withTLS specifies whether to use TLS when connecting to the targetAddr.
	withTLS bool
	// proto specifies the tunnel protocol (e.g., proto.ProtoHTTP, proto.ProtoTCP).
	proto uint8
	// name is the desired subdomain for the tunnel (e.g., "example" for "example.domain.com").
	name string
	// metrics specifies if the client want to stream the tunnel metrics.
	metrics bool
	// httpLog species if the client want to stream http logs.
	httpLog bool
	// domain specifies the approved requested domain name.
	domain string
	// apiKey specifies the server-issued JWT token.
	apiKey string
	// ttl specifies the time-to-live of this client.
	ttl uint64
	// authType is the authentication type for the tunnel.
	authType uint8
	// authUsername is the username for HTTP basic auth.
	authUsername string
	// authPassword is the password for HTTP basic auth.
	authPassword string
	// authToken is the bearer token for bearer token auth.
	authToken string
	// allowHTTP specifies if this tunnel allows HTTP requests, ignored if tunnel protocol is HTTP.
	allowHTTP bool
	// metricsch i used to send http logs and metrics to bubbletea application.
	metricsch chan<- any
	metricsmu sync.Mutex
}

// New creates a new Client with the specified configuration options.
func New(opts ...OptFunc) (*Client, error) {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}

	if c.addr == "" {
		return nil, errors.New("addr must be set")
	}
	if c.targetAddr == "" {
		return nil, errors.New("targetAddr must be set")
	}
	if c.name == "" {
		return nil, errors.New("name must be set")
	}
	if c.proto == 0 {
		return nil, errors.New("proto must be set")
	}

	return c, nil
}

func (c *Client) RunWithTCP(ctx context.Context) error {
	dialer := net.Dialer{}

	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}
	defer conn.Close()

	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = false

	session, err := yamux.Client(conn, yamuxConfig)
	if err != nil {
		return fmt.Errorf("failed to create yamux client: %w", err)
	}
	defer session.Close()

	stream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open yamux session: %w", err)
	}
	defer stream.Close()

	responseHeader, err := c.sendRequest(ctx, stream)
	if err != nil {
		return err
	}

	switch responseHeader.Type {
	case proto.TypeResponse:
		// OK
	case proto.TypeError:
		buf := make([]byte, responseHeader.Length)
		_, err = io.ReadFull(stream, buf)
		if err != nil {
			return fmt.Errorf("failed to read error payload: %w", err)
		}
		return fmt.Errorf("server error: %s", string(buf))
	default:
		return fmt.Errorf("unexpected header type: %v", responseHeader.Type)
	}

	buf := make([]byte, responseHeader.Length)
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return fmt.Errorf("failed to read response payload: %w", err)
	}

	response, err := proto.DeserializeResponse(buf)
	if err != nil {
		return fmt.Errorf("failed to deserialize response: %w", err)
	}

	switch response.Status {
	case proto.StatusNameTaken:
		prettyPrint("err", fmt.Sprintf("subdomain '%s' is already in use", c.name))
		return ErrNameTaken
	case proto.StatusUnsupportedProto:
		prettyPrint("err", fmt.Sprintf("protocol '%v' is not supported", c.proto))
		return ErrUnsupportedProto
	case proto.StatusOK:
	default:
		prettyPrint("err", fmt.Sprintf("unexpected response status: %v", response.Status))
		return fmt.Errorf("unexpected response status: %v", response.Status)
	}

	expiresAt := time.Now().Add(time.Duration(response.TTLHours))
	prettyPrint(
		"inf",
		"tunnel created!",
		fmt.Sprintf("%s%s", Proto(c.proto), response.Domain),
		fmt.Sprintf("tunnel expires at %s", expiresAt.Format("Jan 2, 2006 3:04 PM")),
	)

	return c.handleMessages(ctx, session)
}

// Run initiates a tunnel handshake with the Wormhole server and manages incoming connections.
func (c *Client) Run(ctx context.Context) error {
	host, _, err := net.SplitHostPort(c.addr)
	if err != nil {
		return fmt.Errorf("failed to parse server address: %w", err)
	}

	tlsConfig := &tls.Config{
		ServerName: host,
	}

	dialer := &tls.Dialer{
		Config: tlsConfig,
	}

	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}
	defer conn.Close()

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return fmt.Errorf("failed to create yamux client: %w", err)
	}
	defer session.Close()

	stream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open yamux session: %w", err)
	}
	defer stream.Close()

	responseHeader, err := c.sendRequest(ctx, stream)
	if err != nil {
		return err
	}

	switch responseHeader.Type {
	case proto.TypeResponse:
		// OK
	case proto.TypeError:
		buf := make([]byte, responseHeader.Length)
		_, err = io.ReadFull(stream, buf)
		if err != nil {
			return fmt.Errorf("failed to read error payload: %w", err)
		}
		return fmt.Errorf("server error: %s", string(buf))
	default:
		return fmt.Errorf("unexpected header type: %v", responseHeader.Type)
	}

	buf := make([]byte, responseHeader.Length)
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return fmt.Errorf("failed to read response payload: %w", err)
	}

	response, err := proto.DeserializeResponse(buf)
	if err != nil {
		return fmt.Errorf("failed to deserialize response: %w", err)
	}

	switch response.Status {
	case proto.StatusNameTaken:
		prettyPrint("err", fmt.Sprintf("subdomain '%s' is already in use", c.name))
		return nil
	case proto.StatusUnsupportedProto:
		prettyPrint("err", fmt.Sprintf("protocol '%v' is not supported", c.proto))
		return nil
	case proto.StatusOK:
	default:
		prettyPrint("err", fmt.Sprintf("unexpected response status: %v", response.Status))
		return fmt.Errorf("unexpected response status: %v", response.Status)
	}

	c.domain = response.Domain
	expiresAt := time.Now().Add(time.Duration(response.TTLHours))
	prettyPrint(
		"inf",
		"tunnel created!",
		fmt.Sprintf("%s%s", Proto(c.proto), response.Domain),
		fmt.Sprintf("tunnel expires at %s", expiresAt.Format("Jan 2, 2006 3:04 PM")),
	)

	return c.handleMessages(ctx, session)
}

// sendRequest sends a request to the Wormhole server and reads the response header.
func (c *Client) sendRequest(ctx context.Context, stream net.Conn) (*proto.Header, error) {
	if deadline, ok := ctx.Deadline(); ok {
		stream.SetDeadline(deadline)
	} else {
		stream.SetDeadline(time.Now().Add(5 * time.Second))
	}

	request := proto.NewRequest(c.proto, c.name, c.ttl, c.apiKey)
	request.AuthType = c.authType
	request.AuthUsername = c.authUsername
	request.AuthPassword = c.authPassword
	request.AuthToken = c.authToken

	serializedRequest, err := proto.SerializeRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	header := proto.NewHeader(proto.TypeRequest, uint64(len(serializedRequest)))
	if c.metrics {
		header.SetFlag(proto.FlagMetrics)
	}
	if c.allowHTTP {
		header.SetFlag(proto.FlagAllowHTTP)
	}
	if c.httpLog {
		header.SetFlag(proto.FlagHTTPLog)
	}

	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize header: %w", err)
	}

	_, err = stream.Write(serializedHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	_, err = stream.Write(serializedRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	buf := make([]byte, proto.HeaderSize)
	_, err = io.ReadFull(stream, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	responseHeader, err := proto.DeserializeHeader(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response header: %w", err)
	}

	return responseHeader, nil
}

// ForwardStream forwards a multiplexed stream from the server to the target address.
func (c *Client) ForwardStream(ctx context.Context, stream net.Conn) error {
	var localConn net.Conn
	var err error

	if c.withTLS {
		localConn, err = (&tls.Dialer{
			Config: &tls.Config{
				InsecureSkipVerify: true,
			},
		}).DialContext(ctx, "tcp", c.targetAddr)
		if err != nil {
			return fmt.Errorf("failed to dial tls target address: %w", err)
		}
	} else {
		localConn, err = (&net.Dialer{}).DialContext(ctx, "tcp", c.targetAddr)
		if err != nil {
			return fmt.Errorf("failed to dial tcp target address: %w", err)
		}
	}

	return proxy.StreamWithContext(ctx, localConn, stream)
}

// handleMessages processes incoming multiplexed streams (control streams) from the server.
func (c *Client) handleMessages(ctx context.Context, session *yamux.Session) error {
	cancelCtx, cancel := context.WithCancel(ctx)

	go func() {
		<-cancelCtx.Done()
		session.Close()
	}()

	for {
		stream, err := session.Accept()
		if err != nil {
			cancel()
			if cancelCtx.Err() != nil {
				return cancelCtx.Err()
			}
			return err
		}

		buf := make([]byte, proto.HeaderSize)
		_, err = io.ReadFull(stream, buf)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				log.Debug().Err(err).Msg("stream connection closed")
				stream.Close()
				continue
			}
			log.Warn().Err(err).Msg("failed to read stream header")
			stream.Close()
			continue
		}

		header, err := proto.DeserializeHeader(buf)
		if err != nil {
			log.Error().Err(err).Msg("failed to deserialize header")
			stream.Close()
			continue
		}

		switch header.Type {
		case proto.TypePing:
			go handlePingStream(cancelCtx, stream)
		case proto.TypeAccess:
			go func(ctx context.Context, stream net.Conn) {
				defer stream.Close()
				err = c.ForwardStream(cancelCtx, stream)
				if isDialError(err) && ctx.Err() == nil {
					fmt.Printf("wormhole [err] %s\n", err.Error())
					// for some reason, cursor is at the end of previous line
					// clear the line and move cursor to start
					fmt.Print("\033[2K\r")
					cancel()
				}
			}(cancelCtx, stream)
		case proto.TypeMetrics:
			c.metricsmu.Lock()
			if c.metricsch == nil {
				program, metricsch := StartMetricsDisplay(c.domain, c.metrics, c.httpLog)
				defer close(metricsch)
				go func() {
					if _, err := program.Run(); err != nil {
						log.Error().Err(err).Msg("metrics display error")
					}
					cancel()
				}()
				c.metricsch = metricsch
			}
			c.metricsmu.Unlock()
			go c.handleMetrics(cancelCtx, header, stream)
		case proto.TypeHTTPLog:
			c.metricsmu.Lock()
			if c.metricsch == nil {
				program, metricsch := StartMetricsDisplay(c.domain, c.metrics, c.httpLog)
				defer close(metricsch)
				go func() {
					if _, err := program.Run(); err != nil {
						log.Error().Err(err).Msg("metrics display error")
					}
					cancel()
				}()
				c.metricsch = metricsch
			}
			c.metricsmu.Unlock()
			go c.handleHTTPLog(cancelCtx, header, stream)
		case proto.TypeEnd:
			stream.Close()
			cancel()
			prettyPrint("inf", "tunnel timed out")
			return nil
		default:
			stream.Close()
		}
	}
}

func isDialError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) && netErr.Op == "dial" {
		return true
	}

	return false
}

// handleHTTPLog handles incoming HTTP log entries from the server.
func (c *Client) handleHTTPLog(ctx context.Context, header *proto.Header, stream net.Conn) error {
	defer stream.Close()

	buf := make([]byte, header.Length)
	_, err := io.ReadFull(stream, buf)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		return fmt.Errorf("failed to read http log: %w", err)
	}

	// This is the "READY" log, should be ignored.
	_, err = proto.DeserializeHTTPLog(buf)
	if err != nil {
		log.Error().Err(err).Msg("failed to deserialize http log")
		return fmt.Errorf("failed to deserialize http log: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			headerBuf := make([]byte, proto.HeaderSize)
			_, err := io.ReadFull(stream, headerBuf)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return nil
				}
				return fmt.Errorf("failed to read http log header: %w", err)
			}

			header, err = proto.DeserializeHeader(headerBuf)
			if err != nil {
				return fmt.Errorf("failed to deserialize header: %w", err)
			}

			if header.Type != proto.TypeHTTPLog {
				log.Warn().Uint8("type", header.Type).Msg("unexpected message type in http log stream")
				return nil
			}

			buf := make([]byte, header.Length)
			_, err = io.ReadFull(stream, buf)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return nil
				}
				return fmt.Errorf("failed to read http log: %w", err)
			}

			httpLog, err := proto.DeserializeHTTPLog(buf)
			if err != nil {
				log.Error().Err(err).Msg("failed to deserialize http log")
				return fmt.Errorf("failed to deserialize http log: %w", err)
			}

			c.metricsch <- HTTPLogMsg{
				Method:    httpLog.Method,
				Path:      httpLog.Path,
				Duration:  httpLog.Duration,
				Timestamp: httpLog.Timestamp,
				Status:    httpLog.Status,
			}
		}
	}
}

// handlePingStream handle server pings using a dedicated stream.
func handlePingStream(ctx context.Context, stream net.Conn) {
	defer stream.Close()

	pongHeader := proto.NewHeader(proto.TypePong, 0)
	serialized, err := proto.SerializeHeader(pongHeader)
	if err != nil {
		log.Error().Err(err).Msg("failed to serialize pong")
		return
	}

	_, err = stream.Write(serialized)
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf := make([]byte, proto.HeaderSize)
			_, err := io.ReadFull(stream, buf)
			if err != nil {
				return
			}

			header, err := proto.DeserializeHeader(buf)
			if err != nil {
				log.Error().Err(err).Msg("failed to deserialize ping header")
				return
			}

			if header.Type != proto.TypePing {
				log.Warn().Uint8("type", header.Type).Msg("unexpected message type in ping stream")
				return
			}

			pongHeader := proto.NewHeader(proto.TypePong, 0)
			serialized, err := proto.SerializeHeader(pongHeader)
			if err != nil {
				log.Error().Err(err).Msg("failed to serialize pong")
				return
			}

			_, err = stream.Write(serialized)
			if err != nil {
				return
			}
		}
	}
}

// handleMetrics handles metrics display.
func (c *Client) handleMetrics(ctx context.Context, header *proto.Header, stream net.Conn) error {
	defer stream.Close()

	buf := make([]byte, header.Length)
	_, err := io.ReadFull(stream, buf)
	if err != nil {
		return fmt.Errorf("failed to read metrics: %w", err)
	}

	deserializedMetrics, err := proto.DeserializeMetrics(buf)
	if err != nil {
		return fmt.Errorf("failed to deserialize metrics: %w", err)
	}

	c.metricsch <- MetricsMsg{
		Ingress:           deserializedMetrics.Ingress,
		Egress:            deserializedMetrics.Egress,
		Uptime:            deserializedMetrics.Uptime,
		ConnectionCount:   deserializedMetrics.ConnectionCount,
		ActiveConnections: deserializedMetrics.ActiveConnections,
		RTT:               deserializedMetrics.RTT,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			headerBuf := make([]byte, proto.HeaderSize)
			_, err := io.ReadFull(stream, headerBuf)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return nil
				}
				return fmt.Errorf("failed to read metrics header: %w", err)
			}

			h, err := proto.DeserializeHeader(headerBuf)
			if err != nil {
				return fmt.Errorf("failed to deserialize header: %w", err)
			}

			metricsBuf := make([]byte, h.Length)
			_, err = io.ReadFull(stream, metricsBuf)
			if err != nil {
				return fmt.Errorf("failed to read metrics: %w", err)
			}

			deserializedMetrics, err := proto.DeserializeMetrics(metricsBuf)
			if err != nil {
				return fmt.Errorf("failed to deserialize metrics: %w", err)
			}

			c.metricsch <- MetricsMsg{
				Ingress:           deserializedMetrics.Ingress,
				Egress:            deserializedMetrics.Egress,
				Uptime:            deserializedMetrics.Uptime,
				ConnectionCount:   deserializedMetrics.ConnectionCount,
				ActiveConnections: deserializedMetrics.ActiveConnections,
				RTT:               deserializedMetrics.RTT,
			}
		}
	}
}

// Proto converts a protocol constant to its string representation.
func Proto(p uint8) string {
	switch p {
	case proto.ProtoHTTP:
		return "https://"
	case proto.ProtoTCP:
		return "tcp:"
	default:
		return ""
	}
}

// prettyPrint logs messages to the console with a specified log level.
func prettyPrint(level string, args ...string) {
	for _, arg := range args {
		fmt.Printf("wormhole [%s] %s\n", level, arg)
	}
}
