package wormhole

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/yamux"
)

type client struct {
	id           string
	wormholeAddr string
	localAddr    string
	targetAddr   string
	proto        string
	cancel       context.CancelFunc
	ctx          context.Context
	logger       Logger
}

func NewClient(id, wormholeAddr, localAddr, targetAddr, proto string) *client {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("could not determine home directory for logging")
	}

	name := "client"
	logPath := filepath.Join(home, "wormhole", name, "logs", "log.txt")

	logger := NewLogger()
	logger.InitMultiWriter(name, logPath)

	return &client{
		id:           id,
		wormholeAddr: wormholeAddr,
		localAddr:    localAddr,
		targetAddr:   targetAddr,
		proto:        proto,
		logger:       logger,
	}
}

func (c *client) Stop() {
	if c.cancel != nil {
		c.cancel()
		c.logger.Info("wormhole client stopped")
		return
	}

	c.logger.Warn("c.Stop() is called but c.cancel is nil")
}

func (c *client) Start(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	conn, err := tls.Dial("tcp", c.wormholeAddr, &tls.Config{
		ServerName: "wormhole.dyastin.tech",
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDialTCP, err)
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToCreateYamuxClient, err)
	}

	go func() {
		<-c.ctx.Done()
		c.logger.Info("session closed")
		session.Close()
	}()

	stream, err := session.Open()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToOpenStream, err)
	}

	msg, err := c.handshake(stream)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHandshakeFailed, err)
	}

	if msg.Status != 0 {
		log.Println(msg.Err)
		return fmt.Errorf("%w: %v", ErrHandshakeFailed, ErrIDAlreadyUsed)
	}

	c.logger.Info("service started")

	for {
		stream, err := session.Accept()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return c.ctx.Err()
			default:
				if errors.Is(session.GoAway(), err) || errors.Is(err, io.EOF) {
					return nil
				}

				c.logger.Error(fmt.Sprintf("%s: %s", ErrFailedToAcceptConn.Error(), err))
				continue
			}
		}

		go func(s net.Conn) {
			if err := c.handleConn(s); err != nil {
				c.logger.Error(fmt.Sprintf("%v\n", err))
			}
		}(stream)
	}
}

// implements simple handshake
func (c *client) handshake(stream net.Conn) (*message, error) {
	defer stream.Close()

	enc := json.NewEncoder(stream)
	dec := json.NewDecoder(stream)

	msg := &message{
		ID:    c.id,
		Proto: c.proto,
	}

	err := enc.Encode(msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToEncodeMessage, err)
	}

	err = dec.Decode(&msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToDecodeMessage, err)
	}

	return msg, nil
}

func (c *client) handleConn(stream net.Conn) error {
	defer stream.Close()

	switch c.proto {
	case ProtoHTTP:
		return c.http(stream)
	case ProtoTCP:
		return c.tcp(stream)
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
		return fmt.Errorf("%w: %v", ErrFailedToDialTCP, err)
	}

	err = req.Write(conn)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToWriteHTTPTunnelRequest, err)
	}

	localBufr := bufio.NewReader(conn)

	resp, err := http.ReadResponse(localBufr, req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToReadTCPTunnelResponse, err)
	}
	defer resp.Body.Close()

	err = resp.Write(stream)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToWriteHTTPTunnelResponse, err)
	}

	return nil
}

func (c *client) tcp(stream net.Conn) error {
	return nil
}
