package wormhole

import "net"

type mockSession struct {
	conn net.Conn
	err  error
}

func (m *mockSession) Open() (net.Conn, error) {
	return m.conn, m.err
}
