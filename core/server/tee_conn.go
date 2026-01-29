// License notice:
//
// Partially modified code are from:
// github.com/inconshreveable/go-vhost
// Copyright 2022 Alan Shreve (@inconshreveable)
// Licensed under the Apache License, Version 2.0
// http://www.apache.org/licenses/LICENSE-2.0

package server

import (
	"bytes"
	"io"
	"net"
	"sync"
)

const (
	bufSize = 1024
)

type TeeConn struct {
	sync.Mutex
	net.Conn
	buf *bytes.Buffer
}

func NewTeeConn(conn net.Conn) (*TeeConn, io.Reader) {
	c := &TeeConn{
		Conn: conn,
		buf:  bytes.NewBuffer(make([]byte, 0, bufSize)),
	}

	return c, io.TeeReader(conn, c.buf)
}

func (c *TeeConn) Read(p []byte) (n int, err error) {
	c.Lock()
	if c.buf == nil {
		c.Unlock()
		return c.Conn.Read(p)
	}
	n, err = c.buf.Read(p)

	if err == io.EOF {
		c.buf = nil

		var n2 int
		n2, err = c.Conn.Read(p[n:])

		n += n2
	}
	c.Unlock()
	return
}
