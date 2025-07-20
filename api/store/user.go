package store

import (
	"github.com/Dyastin-0/wormhole/api/db"
	"golang.org/x/net/context"
)

type userStore struct {
	query *db.Queries
}

func NewUserStore(q *db.Queries) *userStore {
	return &userStore{
		query: q,
	}
}

func (u *userStore) Create(ctx context.Context, req *db.CreateUserParams) (*db.User, error) {
	res, err := u.query.CreateUser(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (u *userStore) GetByEmail(ctx context.Context, email string) (*db.User, error) {
	res, err := u.query.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (u *userStore) Get(ctx context.Context, id string) (*db.User, error) {
	res, err := u.query.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (u *userStore) GetWithEmailAndPassword(ctx context.Context, req *db.GetUserByEmailAndPasswordParams) (*db.User, error) {
	res, err := u.query.GetUserByEmailAndPassword(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (u *userStore) Update(ctx context.Context, req *db.UpdateUserParams) (*db.User, error) {
	res, err := u.query.UpdateUser(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (u *userStore) Delete(ctx context.Context, id string) (*db.User, error) {
	res, err := u.query.DeleteUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
