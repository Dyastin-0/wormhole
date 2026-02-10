package stream

import (
	"bufio"
	"net"
)

type BuffConn struct {
	net.Conn
	r *bufio.Reader
}

func (bc *BuffConn) Read(p []byte) (int, error) {
	return bc.r.Read(p)
}

func (bc *BuffConn) Write(b []byte) (int, error) {
	return bc.Conn.Write(b)
}
