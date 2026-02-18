package auth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// BearerAuth implements bearer token auth.
type BearerAuth struct {
	token string
	realm string
}

func NewBearerAuth(token string) (*BearerAuth, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	return &BearerAuth{
		token: token,
		realm: "Wormhole Tunnel",
	}, nil
}

func (b *BearerAuth) Authenticate(req *http.Request) bool {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		log.Debug().Msg("no authorization header")
		return false
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		log.Debug().Msg("not bearer auth")
		return false
	}

	token := strings.TrimSpace(auth[7:])

	return subtle.ConstantTimeCompare(
		[]byte(token),
		[]byte(b.token),
	) == 1
}

func (b *BearerAuth) Realm() string {
	return b.realm
}

func (b *BearerAuth) Method() string {
	return MethodBearer
}
