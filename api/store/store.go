// Package store handle DB queries
package store

import (
	"github.com/Dyastin-0/wormhole/api/db"
)

type Store struct {
	User   *userStore
	Tunnel *tunnelStore
	Auth   *authStore
}

func New(q *db.Queries) *Store {
	return &Store{
		User:   NewUserStore(q),
		Tunnel: NewTunnelStore(q),
		Auth:   NewAuthStore(q),
	}
}
