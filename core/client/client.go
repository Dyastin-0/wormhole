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
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/proxy"
	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog/log"
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
	// domain specifies the approved requested domain name.
	domain string
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
	yamuxConfig.MaxStreamWindowSize = 16 * 1024 * 1024
	yamuxConfig.KeepAliveInterval = 1 * time.Second

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
		return nil
	case proto.StatusUnsupportedProto:
		prettyPrint("err", fmt.Sprintf("protocol '%v' is not supported", c.proto))
		return nil
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

	request := proto.NewRequest(c.proto, c.name)
	serializedRequest, err := proto.SerializeRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	header := proto.NewHeader(proto.TypeRequest, uint64(len(serializedRequest)))
	if c.metrics {
		header.SetFlag(proto.FlagMetrics)
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
		localConn, err = (&tls.Dialer{}).DialContext(ctx, "tcp", c.targetAddr)
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
	go func() {
		<-ctx.Done()
		session.Close()
	}()

	for {
		stream, err := session.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
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
		case proto.TypeAccess:
			go func(ctx context.Context, stream net.Conn) {
				defer stream.Close()
				if err := c.sendAck(stream); err != nil {
					log.Error().Err(err).Msg("failed to send ack")
					return
				}
				c.ForwardStream(ctx, stream)
			}(ctx, stream)
		case proto.TypeMetrics:
			go c.handleMetrics(ctx, header, stream)
		case proto.TypeEnd:
			stream.Close()
			prettyPrint("inf", "tunnel timed out")
			return nil
		default:
			log.Debug().Msgf("unexpected header type: %v", header.Type)
			stream.Close()
		}
	}
}

// handleMetrics handles metrics display.
func (c *Client) handleMetrics(ctx context.Context, header *proto.Header, stream net.Conn) error {
	defer stream.Close()

	program, metricsChan := StartMetricsDisplay(c.domain)
	defer close(metricsChan)

	go func() {
		if _, err := program.Run(); err != nil {
			log.Error().Err(err).Msg("metrics display error")
		}
	}()

	buf := make([]byte, header.Length)
	_, err := io.ReadFull(stream, buf)
	if err != nil {
		return fmt.Errorf("failed to read metrics: %w", err)
	}

	deserializedMetrics, err := proto.DeserializeMetrics(buf)
	if err != nil {
		return fmt.Errorf("failed to deserialize metrics: %w", err)
	}

	metricsChan <- MetricsMsg{
		Ingress:           deserializedMetrics.Ingress,
		Egress:            deserializedMetrics.Egress,
		Uptime:            deserializedMetrics.Uptime,
		ConnectionCount:   deserializedMetrics.ConnectionCount,
		ActiveConnections: deserializedMetrics.ActiveConnections,
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

			metricsChan <- MetricsMsg{
				Ingress:           deserializedMetrics.Ingress,
				Egress:            deserializedMetrics.Egress,
				Uptime:            deserializedMetrics.Uptime,
				ConnectionCount:   deserializedMetrics.ConnectionCount,
				ActiveConnections: deserializedMetrics.ActiveConnections,
			}
		}
	}
}

// sendAck sends an acknowledgment for a TypeAccess message.
func (c *Client) sendAck(stream net.Conn) error {
	header := proto.NewHeader(proto.TypeAck, 0)
	serializedHeader, err := proto.SerializeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to serialize ack header: %w", err)
	}

	_, err = stream.Write(serializedHeader)
	if err != nil {
		return fmt.Errorf("failed to write ack: %w", err)
	}

	return nil
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
