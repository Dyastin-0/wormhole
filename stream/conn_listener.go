package stream

import (
	"fmt"
	"net"
)

type ConnListener struct {
	conn net.Conn
	done chan struct{}
}

func NewConnListener(conn net.Conn) *ConnListener {
	return &ConnListener{
		conn: conn,
		done: make(chan struct{}),
	}
}

func (l *ConnListener) Accept() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, fmt.Errorf("done")
	default:
		fmt.Println("accept: returning conn")
		close(l.done)
		return l.conn, nil
	}
}

func (l *ConnListener) Close() error {
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return nil
}

func (l *ConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
