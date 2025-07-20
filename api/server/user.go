package server

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/api/hash"
	userpb "github.com/Dyastin-0/wormhole/api/proto/user"
	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var defaultHash = hash.New()

type UserServer struct {
	userpb.UnimplementedUserServiceServer
	store *store.Store
	hash  hash.Hash
}

func NewUserServer(store *store.Store) *UserServer {
	return &UserServer{
		store: store,
		hash:  defaultHash,
	}
}

func (s *UserServer) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.User, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgEmail)
	}

	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgPassword)
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgName)
	}

	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgUsername)
	}

	id := uuid.New().String()

	byteHashedPassword, err := s.hash.Generate([]byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGenerateHash)
	}

	reqq := &db.CreateUserParams{
		ID:       id,
		Email:    req.Email,
		Name:     req.Name,
		Username: req.Username,
		Password: string(byteHashedPassword),
	}

	res, err := s.store.User.Create(ctx, reqq)
	if err != nil {
		if errors.Is(err, sqlite3.ErrConstraintUnique) {
			return nil, status.Errorf(codes.InvalidArgument, "%s: %s", ErrFailedToCreateUser, "email or username is already used")
		}
		return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToCreateUser, err)
	}

	return &userpb.User{
		Id:       res.ID,
		Email:    res.Email,
		Username: res.Username,
		Name:     res.Name,
	}, nil
}

func (s *UserServer) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.User, error) {
	user, err := s.store.User.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, ErrUserNotFound)
		}
		return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToGetUser, err)
	}

	return &userpb.User{
		Id:       user.ID,
		Email:    user.Email,
		Username: user.Name,
	}, nil
}

func (s *UserServer) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.User, error) {
	if req.Id == "" {
		return nil, status.Error(codes.Internal, ErrMissingArgID)
	}

	if req.Password != "" {
		byteHashedPassword, err := s.hash.Generate([]byte(req.Password))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToGenerateHash, err)
		}

		req.Password = string(byteHashedPassword)
	}

	param := &db.UpdateUserParams{
		ID:       toNullString(req.Id),
		Email:    toNullString(req.Email),
		Name:     toNullString(req.Name),
		Password: toNullString(req.Password),
	}

	res, err := s.store.User.Update(ctx, param)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToUpdateUser, err)
	}

	return &userpb.User{
		Id:       res.ID,
		Email:    res.Email,
		Username: res.Name,
	}, nil
}

func (s *UserServer) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.User, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgID)
	}

	res, err := s.store.User.Delete(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToDeleteUser, err)
	}

	return &userpb.User{
		Id:       res.ID,
		Email:    res.Email,
		Username: res.Name,
	}, nil
}

func toNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}
