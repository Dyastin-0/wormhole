// Package auth implements an authenticator (basic auth) for HTTP tunnels, optionally.
package auth

import (
	"net"
	"net/http"
)

// Authenticator defines the interface for authentication methods.
type Authenticator interface {
	// Authenticate verifies the request and returns true if authenticated.
	Authenticate(req *http.Request) bool

	// SendChallenge sends the appropriate authentication challenge response.
	SendChallenge(conn net.Conn)

	// IsEnabled returns true if authentication is configured.
	IsEnabled() bool
}

// NoAuth represents no authentication (public access)
type NoAuth struct{}

func (n *NoAuth) Authenticate(req *http.Request) bool {
	return true
}

func (n *NoAuth) SendChallenge(conn net.Conn) {
	// No challenge needed for public access
}

func (n *NoAuth) IsEnabled() bool {
	return false
}
