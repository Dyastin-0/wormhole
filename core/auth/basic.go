package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

// BasicAuth implements HTTP basic auth with bcrypt password hashing.
type BasicAuth struct {
	username string
	password string
	realm    string
}

func NewBasicAuth(username, password string) (*BasicAuth, error) {
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}

	return &BasicAuth{
		username: username,
		password: password,
		realm:    "Wormhole Tunnel",
	}, nil
}

func (b *BasicAuth) Authenticate(req *http.Request) bool {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return false
	}

	credentials := strings.SplitN(string(payload), ":", 2)
	if len(credentials) != 2 {
		return false
	}

	username := credentials[0]
	password := credentials[1]

	usernameMatch := subtle.ConstantTimeCompare(
		[]byte(username),
		[]byte(b.username),
	) == 1

	passwordMatch := subtle.ConstantTimeCompare(
		[]byte(password),
		[]byte(b.password),
	) == 1

	authenticated := usernameMatch && passwordMatch

	return authenticated
}

func (b *BasicAuth) Realm() string {
	return b.realm
}

func (b *BasicAuth) Method() string {
	return MethodBasic
}
