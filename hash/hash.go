// Package hash implements Hash using bcrypt used in auth service
package hash

import "golang.org/x/crypto/bcrypt"

type Hash interface {
	Generate([]byte) ([]byte, error)
	Compare([]byte, []byte) error
}

type hash struct{}

func New() *hash {
	return &hash{}
}

func (h *hash) Generate(pw []byte) ([]byte, error) {
	bytehash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return bytehash, nil
}

func (h *hash) Compare(hash, pw []byte) error {
	return bcrypt.CompareHashAndPassword(hash, pw)
}
