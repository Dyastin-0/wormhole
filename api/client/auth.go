package client

import (
	"context"
	"fmt"

	"github.com/Dyastin-0/wormhole/api/proto/auth"
	"google.golang.org/grpc"
)

type AuthClient struct {
	service auth.AuthServiceClient
}

func NewAuthClient(addr string, opts ...grpc.DialOption) (*AuthClient, error) {
	conn, err := New(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create an auth client: %v", err)
	}

	ac := &AuthClient{
		service: auth.NewAuthServiceClient(conn),
	}

	return ac, nil
}

func (a *AuthClient) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.AuthResponse, error) {
	return a.service.Register(ctx, req)
}

func (a *AuthClient) Login(ctx context.Context, req *auth.LoginRequest) (*auth.AuthResponse, error) {
	return a.service.Login(ctx, req)
}

func (a *AuthClient) Logout(ctx context.Context, req *auth.LogoutRequest) (*auth.LogoutResponse, error) {
	return a.service.Logout(ctx, req)
}

func (a *AuthClient) Refresh(ctx context.Context, req *auth.RefreshRequest) (*auth.AuthResponse, error) {
	return a.service.Refresh(ctx, req)
}
