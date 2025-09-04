package store

import "github.com/Dyastin-0/wormhole/api/db"

type APIKeyStore struct {
	query *db.Queries
}

func NewAPIKeyStore(q *db.Queries) *APIKeyStore {
	return &APIKeyStore{
		query: q,
	}
}

func (a *APIKeyStore) Revoke() {
}
