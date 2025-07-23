// Package token implements a token issuer with custom payload for auth
package token

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Issuer struct {
	accessSecret  string
	refreshSecret string
	apiSecret     string
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	PayloadEmail = "email"
	PayloadID    = "id"
	PayloadRole  = "role"

	KeyRefreshToken = "refresh_token"

	DefaultAccessTTL  = 15 * time.Minute
	DefaultRefreshTTL = 24 * time.Hour
)

func DefaultIssuer() *Issuer {
	accessSecret := os.Getenv("ACCESS_SECRET")
	refreshSecret := os.Getenv("REFRESH_SECRET")

	return New(accessSecret, refreshSecret)
}

func New(accessSecret, refreshSecret string) *Issuer {
	return &Issuer{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
	}
}

func (i *Issuer) NewAccessToken(id, email, role string, ttl time.Duration) (string, error) {
	return newToken(id, email, role, i.accessSecret, ttl)
}

func (i *Issuer) ParseAccessToken(token string) (*jwt.MapClaims, error) {
	return parseToken(token, i.accessSecret)
}

func (i *Issuer) NewRefreshToken(id, email, role string, ttl time.Duration) (string, error) {
	return newToken(id, email, role, i.refreshSecret, ttl)
}

func (i *Issuer) ParseRefreshToken(token string) (*jwt.MapClaims, error) {
	return parseToken(token, i.refreshSecret)
}

func newToken(id, email, role, secret string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"email": email,
		"id":    id,
		"role":  role,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(ttl).Unix(),
		"iss":   "wormhole",
		"jti":   uuid.NewString(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func parseToken(token, secret string) (*jwt.MapClaims, error) {
	claims := &jwt.MapClaims{}

	_, err := jwt.ParseWithClaims(
		token,
		claims,
		func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		},
	)
	if err != nil {
		return nil, err
	}

	if _, ok := (*claims)[PayloadEmail].(string); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing email payload")
	}

	if _, ok := (*claims)[PayloadRole].(string); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing role payload")
	}

	if _, ok := (*claims)[PayloadID].(string); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing id payload")
	}

	return claims, nil
}
