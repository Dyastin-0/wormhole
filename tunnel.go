package wormhole

import "github.com/hashicorp/yamux"

type tunnel struct {
	proto   string
	session *yamux.Session
}
