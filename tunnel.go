package wormhole

import (
	"net"
)

type session interface {
	Open() (net.Conn, error)
}

type tunnel struct {
	id      string
	userID  string
	proto   string
	session session
}
