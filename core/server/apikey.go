package server

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when a token is malformed or invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when a token has expired.
	ErrTokenExpired = errors.New("token has expired")
	// ErrInvalidClaims is returned when token claims are invalid.
	ErrInvalidClaims = errors.New("invalid token claims")
)

// APIKeyClaims represents the JWT claims for an API key.
type APIKeyClaims struct {
	TTL uint64 `json:"ttl"`
	jwt.RegisteredClaims
}

// APIKeyIssuer handles the creation and validation of JWT-based API keys.
type APIKeyIssuer struct {
	secret []byte
}

// NewAPIKeyIssuer creates a new APIKeyIssuer with the provided secret.
// If secret is nil or empty, a random secret will be generated.
func NewAPIKeyIssuer(secret []byte) (*APIKeyIssuer, error) {
	if len(secret) == 0 {
		var err error
		secret, err = generateSecret(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret: %w", err)
		}
	}

	return &APIKeyIssuer{
		secret: secret,
	}, nil
}

// Issue creates a new API key JWT token with the specified parameters.
func (i *APIKeyIssuer) Issue(ttl uint64, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := &APIKeyClaims{
		TTL: ttl,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "wormhole-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// Validate validates an API key JWT token and returns its claims.
func (i *APIKeyIssuer) Validate(tokenString string) (*APIKeyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &APIKeyClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return i.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*APIKeyClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// generateSecret generates a cryptographically secure random secret.
func generateSecret(length int) ([]byte, error) {
	secret := make([]byte, length)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}
