package wormhole

import (
	"net"
)

type session interface {
	Open() (net.Conn, error)
}

type tunnel struct {
	proto   string
	session session
}
