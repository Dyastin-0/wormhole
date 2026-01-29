package server

import (
	"bufio"
	"net"
)

type BuffConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *BuffConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *BuffConn) Write(p []byte) (int, error) {
	return c.Conn.Write(p)
}

func (c *BuffConn) GetReader() *bufio.Reader {
	return c.r
}
