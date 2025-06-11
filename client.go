package wormhole

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

type client struct {
	id           string
	wormholeAddr string
	localAddr    string
	remoteAddr   string
}

func NewClient(id, wormholeAddr, localAddr, remoteAddr string) *client {
	return &client{
		id:           id,
		wormholeAddr: wormholeAddr,
		localAddr:    localAddr,
		remoteAddr:   remoteAddr,
	}
}

func (c *client) Start() error {
	conn, err := net.Dial("tcp", c.wormholeAddr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDialWormhole, err)
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
		return fmt.Errorf("%w: %v", ErrHandshakeFailed, "id already used")
	}

	log.Println("connection established")

	for {
		stream, err := session.Accept()
		if err != nil {
			log.Println(err)
		}

		go func(s net.Conn) {
			if err := c.handleConn(s); err != nil {
				log.Println(err.Error())
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
		ID: c.id,
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
	// for {
	// wait for and forward request to c.remoteAddr (what proto??)
	// write response to the server (stream)
	// }
	return nil
}
