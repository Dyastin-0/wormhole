package server

import (
	"bufio"
	"net"
)

// PeekableConn wraps a net.Conn to allow peeking without consuming data.
type PeekableConn struct {
	net.Conn
	reader *bufio.Reader
}

// NewPeekableConn returns a new PeekableConn.
func NewPeekableConn(conn net.Conn) *PeekableConn {
	return &PeekableConn{
		Conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

// Read uses the underlying bufio.Reader.
func (pc *PeekableConn) Read(b []byte) (int, error) {
	return pc.reader.Read(b)
}

// Peek uses the underlying bufio.Reader.
func (pc *PeekableConn) Peek(n int) ([]byte, error) {
	return pc.reader.Peek(n)
}
