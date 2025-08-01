package client

import (
	"context"
	"fmt"

	"github.com/Dyastin-0/wormhole/api/proto/user"
	"google.golang.org/grpc"
)

type UserClient struct {
	service user.UserServiceClient
}

func NewUserClient(addr string, opts ...grpc.DialOption) (*UserClient, error) {
	conn, err := New(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create an auth client: %v", err)
	}

	ac := &UserClient{
		service: user.NewUserServiceClient(conn),
	}

	return ac, nil
}

func (u *UserClient) Creat(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	return u.service.CreateUser(ctx, req)
}

func (u *UserClient) Get(ctx context.Context, req *user.GetUserRequest) (*user.User, error) {
	return u.service.GetUser(ctx, req)
}

func (u *UserClient) Update(ctx context.Context, req *user.UpdateUserRequest) (*user.User, error) {
	return u.service.UpdateUser(ctx, req)
}

func (u *UserClient) Delete(ctx context.Context, req *user.DeleteUserRequest) (*user.User, error) {
	return u.service.DeleteUser(ctx, req)
}

func (u *UserClient) GetMany(ctx context.Context, req *user.GetUsersRequest) (*user.GetUsersResponse, error) {
	return u.service.GetUsers(ctx, req)
}
