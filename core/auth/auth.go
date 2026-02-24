// Package auth implements an authenticator (basic auth) for HTTP tunnels, optionally.
package auth

import (
	"net/http"
)

const (
	MethodBearer = "bearer"
	MethodBasic  = "basic"
)

// Authenticator defines the interface for authentication methods.
type Authenticator interface {
	// Authenticate verifies the request and returns true if authenticated.
	Authenticate(req *http.Request) bool
	// Realm returns the realm.
	Realm() string
	// Method returns the authentication method.
	Method() string
}

// NoAuth represents no authentication (public access)
type NoAuth struct{}

func (n *NoAuth) Authenticate(req *http.Request) bool {
	return true
}

func (n *NoAuth) Realm() string  { return "realm" }
func (n *NoAuth) Method() string { return "no-auth" }
