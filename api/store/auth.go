package store

import (
	"github.com/Dyastin-0/wormhole/api/db"
	"golang.org/x/net/context"
)

type authStore struct {
	query *db.Queries
}

func NewAuthStore(q *db.Queries) *authStore {
	return &authStore{
		query: q,
	}
}

func (a *authStore) Delete(ctx context.Context, req *db.DeleteRefreshTokenParams) error {
	err := a.query.DeleteRefreshToken(ctx, *req)
	if err != nil {
		return err
	}

	return nil
}

func (a *authStore) Create(ctx context.Context, req *db.CreateRefreshTokenParams) error {
	err := a.query.CreateRefreshToken(ctx, *req)
	if err != nil {
		return err
	}

	return nil
}

func (a *authStore) Get(ctx context.Context, req *db.GetRefreshTokenParams) (*db.RefreshToken, error) {
	res, err := a.query.GetRefreshToken(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
