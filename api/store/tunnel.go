package store

import (
	"context"

	"github.com/Dyastin-0/wormhole/api/db"
)

type tunnelStore struct {
	query *db.Queries
}

func NewTunnelStore(q *db.Queries) *tunnelStore {
	return &tunnelStore{
		query: q,
	}
}

func (t *tunnelStore) Create(ctx context.Context, req *db.CreateTunnelParams) (*db.Tunnel, error) {
	res, err := t.query.CreateTunnel(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (t *tunnelStore) Get(ctx context.Context, req *db.GetTunnelParams) (*db.Tunnel, error) {
	res, err := t.query.GetTunnel(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (t *tunnelStore) Update(ctx context.Context, req *db.UpdateTunnelParams) (*db.Tunnel, error) {
	res, err := t.query.UpdateTunnel(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (t *tunnelStore) Delete(ctx context.Context, req *db.DeleteTunnelParams) (*db.Tunnel, error) {
	res, err := t.query.DeleteTunnel(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
