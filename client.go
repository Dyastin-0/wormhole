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

	"github.com/Dyastin-0/wormhole/logger"
	"github.com/hashicorp/yamux"
)

type client struct {
	apiKey       string
	name         string
	tunnelID     string
	wormholeAddr string
	targetAddr   string
	proto        string
	cancel       context.CancelFunc
	ctx          context.Context
	Logger       logger.Logger
}

var InsecureTLSConfig = func() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}

func NewClient(api, id, name, wormholeAddr, targetAddr, proto string) *client {
	return &client{
		name:         name,
		tunnelID:     id,
		wormholeAddr: wormholeAddr,
		targetAddr:   targetAddr,
		proto:        proto,
		Logger:       &logger.NoopLogger{},
	}
}

func (c *client) Stop() {
	if c.cancel != nil {
		c.cancel()
		c.Logger.Info("wormhole client stopped")
		return
	}

	c.Logger.Warn("c.Stop() is called but c.cancel is nil")
}

func (c *client) Start(ctx context.Context, tlsConfig *tls.Config) error {
	if tlsConfig == nil {
		return ErrNilTLSConfig
	}

	if ctx == nil {
		return ErrNilContext
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	conn, err := tls.Dial("tcp", c.wormholeAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToDialTCP, err)
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToCreateYamuxClient, err)
	}
	defer session.Close()

	go func() {
		<-c.ctx.Done()
		c.Logger.Info("session closed")
		session.Close()
	}()

	stream, err := session.Open()
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToOpenStream, err)
	}

	msg, err := c.handshake(stream)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrHandshakeFailed, err)
	}

	// handle other messages
	go c.handleMessage(stream)

	if msg.Status != StatusOK {
		c.Logger.Error(msg.Err)
		return fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrTunnelNameAlreadyUsed)
	}

	c.Logger.Info("tunnel started")
	switch c.proto {
	case ProtoHTTP:
		fmt.Printf("wormhole [inf]: address -> https://%s\n", msg.TunnelDomain)
	case ProtoTCP:
		fmt.Printf("wormhole [inf]: address -> %s:8443\n", msg.TunnelDomain)
	}

	for {
		stream, err := session.Accept()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return nil
			default:
				if errors.Is(err, yamux.ErrTimeout) ||
					errors.Is(err, yamux.ErrRemoteGoAway) ||
					errors.Is(err, io.EOF) {
					fmt.Println("wormhole [err]: server connection closed")
					fmt.Println("wormhole [err]: %v", err)
					return nil
				}

				c.Logger.Error(fmt.Sprintf("%s: %s", ErrFailedToAcceptConn.Error(), err))
				continue
			}
		}

		go func(s net.Conn) {
			if err := c.handleConn(s); err != nil {
				fmt.Printf("wormhole [err]: %s\n", err.Error())
				c.Logger.Error(fmt.Sprintf("%v", err))
			}
		}(stream)
	}
}

func (c *client) handleMessage(stream net.Conn) error {
	defer stream.Close()

	dec := json.NewDecoder(stream)

	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()

		default:
			msg := &message{}

			err := dec.Decode(msg)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return err
				}

				fmt.Printf("wormhole [err]: %s\n", ErrFailedToDecodeMessage.Error())
				continue
			}

			switch msg.Message {
			case MsgTunnelttlTimeout:
				c.Stop()
				fmt.Printf("wormhole [inf]: %s\n", msg.Message)
				return nil
			default:
			}
		}
	}
}

// implements simple handshake
func (c *client) handshake(stream net.Conn) (*message, error) {
	enc := json.NewEncoder(stream)
	dec := json.NewDecoder(stream)

	msg := &message{
		APIKey:      c.apiKey,
		TunnelProto: c.proto,
		TunnelID:    c.tunnelID,
		TunnelName:  c.name,
	}

	err := enc.Encode(msg)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err)
	}

	err = dec.Decode(&msg)
	if err != nil {
		return nil, fmt.Errorf("client err: %v: %w", ErrFailedToDecodeMessage, err)
	}

	return msg, nil
}

func (c *client) handleConn(stream net.Conn) error {
	defer stream.Close()

	switch c.proto {
	case ProtoHTTP:
		return c.http(stream)
	case ProtoTCP:
		conn, err := net.Dial("tcp", c.targetAddr)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrFailedToDialTCP, err)
		}

		return c.tcp(stream, conn)
	default:
		return ErrUnsupportedProtocol
	}
}

func (c *client) http(stream net.Conn) error {
	bufr := bufio.NewReader(stream)

	req, err := http.ReadRequest(bufr)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	conn, err := net.Dial("tcp", c.targetAddr)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToDialTCP, err)
	}
	defer conn.Close()

	err = req.Write(conn)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToWriteHTTPRequest, err)
	}

	localBufr := bufio.NewReader(conn)

	resp, err := http.ReadResponse(localBufr, req)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToReadHTTPResponse, err)
	}
	defer resp.Body.Close()

	err = resp.Write(stream)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToWriteHTTPResponse, err)
	}

	return nil
}

func (c *client) tcp(src, dst net.Conn) error {
	errch := make(chan error, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		errch <- c.stream(src, dst)
		wg.Done()
	}()

	go func() {
		errch <- c.stream(dst, src)
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

func (c *client) stream(src, dst net.Conn) error {
	defer dst.Close()

	_, err := io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}
