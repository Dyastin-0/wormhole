package wormhole

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/hashicorp/yamux"
)

type client struct {
	id           string
	wormholeAddr string
	localAddr    string
	targetAddr   string
	proto        string
}

func NewClient(id, wormholeAddr, localAddr, targetAddr, proto string) *client {
	return &client{
		id:           id,
		wormholeAddr: wormholeAddr,
		localAddr:    localAddr,
		targetAddr:   targetAddr,
		proto:        proto,
	}
}

func (c *client) Start() error {
	conn, err := net.Dial("tcp", c.wormholeAddr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDialTCP, err)
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToCreateYamuxClient, err)
	}

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

	for {
		stream, err := session.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func(s net.Conn) {
			if err := c.handleConn(s); err != nil {
				log.Printf("%v\n", err)
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
	case httpProto:
		return c.http(stream)
	case tcpProto:
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
