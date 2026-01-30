package server

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"sync"
)

type BuffConn struct {
	net.Conn
	r  *bufio.Reader
	mu sync.Mutex
	p  *bytes.Buffer
}

func (c *BuffConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.p != nil && c.p.Len() > 0 {
		n, err := c.p.Read(p)
		if err == io.EOF {
			c.p = nil
			if n < len(p) {
				n2, err2 := c.r.Read(p[n:])
				return n + n2, err2
			}
		}
		return n, err
	}

	// Then read from the buffered reader
	return c.r.Read(p)
}

func (c *BuffConn) Write(p []byte) (int, error) {
	return c.Conn.Write(p)
}

func (c *BuffConn) GetReader() *bufio.Reader {
	return c.r
}
