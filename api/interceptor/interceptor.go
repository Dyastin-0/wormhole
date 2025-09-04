// Package interceptor implements custom unary and stream interceptor for auth
package interceptor

import (
	"context"
	"strings"

	"github.com/Dyastin-0/wormhole/api/header"
	"github.com/Dyastin-0/wormhole/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	TokenJWT    = "jwt"
	TokenAPIKey = "apikey"
)

type (
	methods    map[string]roles
	roles      map[string]bool
	CtxPayload string
)

type AuthInterceptor struct {
	methods methods
	issuer  *token.Issuer

	apiKeyEnabled   bool
	apiKeyValidator *APIKeyValidator
}

type APIKeyValidator struct{}

func NewAuthInterceptor(
	methods methods,
	issuer *token.Issuer,
	apiKeyValidator *APIKeyValidator,
) *AuthInterceptor {
	authInterceptor := &AuthInterceptor{
		methods: methods,
		issuer:  issuer,
	}

	if apiKeyValidator != nil {
		authInterceptor.apiKeyValidator = apiKeyValidator
		authInterceptor.apiKeyEnabled = true
	}

	return authInterceptor
}

func DefaultMethods() methods {
	return methods{
		"/auth.AuthService/Logout": roles{token.RoleAdmin: true, token.RoleUser: true},

		"/tunnel.TunnelService/CreateTunnel": roles{token.RoleAdmin: true, token.RoleUser: true},
		"/tunnel.TunnelService/DeleteTunnel": roles{token.RoleAdmin: true, token.RoleUser: true},
		"/tunnel.TunnelService/GetTunnel":    roles{token.RoleAdmin: true, token.RoleUser: true},
		"/tunnel.TunnelService/GetTunnels":   roles{token.RoleAdmin: true, token.RoleUser: true},
		"/tunnel.TunnelService/UpdateTunnel": roles{token.RoleAdmin: true, token.RoleUser: true},

		"/user.UserService/DeleteUser": roles{token.RoleAdmin: true, token.RoleUser: true},
		"/user.UserService/GetUser":    roles{token.RoleAdmin: true, token.RoleUser: true},
		"/user.UserService/GetUsers":   roles{token.RoleAdmin: true, token.RoleUser: true},
		"/user.UserService/UpdateUser": roles{token.RoleAdmin: true, token.RoleUser: true},
		"/user.UserService/CreateUser": roles{token.RoleAdmin: true},
	}
}

func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		allowedRoles, protected := a.methods[info.FullMethod]
		if !protected {
			return handler(ctx, req)
		}

		ctx, err := a.authorize(ctx, allowedRoles)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (a *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		allowedRoles, protected := a.methods[info.FullMethod]
		if !protected {
			return handler(srv, ss)
		}

		ctx, err := a.authorize(ss.Context(), allowedRoles)
		if err != nil {
			return err
		}

		return handler(ctx, ss)
	}
}

func (a *AuthInterceptor) authorize(ctx context.Context, roles map[string]bool) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authToken, tokenType, err := a.getToken(md)
	if err != nil {
		return nil, err
	}

	switch tokenType {
	case TokenJWT:
		payload, err := a.issuer.ParseAccessToken(authToken)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid access token: %s", err)
		}

		rolePayload := (*payload)[token.PayloadRole].(string)

		for role := range roles {
			if role == rolePayload {
				id := (*payload)[token.PayloadID]

				newCtx := context.WithValue(ctx, CtxPayload("user_id"), id)
				return newCtx, nil
			}
		}

	case TokenAPIKey:
	}

	return nil, status.Error(codes.Unauthenticated, "not authorized")
}

func (a *AuthInterceptor) getToken(md metadata.MD) (string, string, error) {
	authToken := md[header.HeaderAuthorization]
	if len(authToken) > 0 {
		authTokenStr := authToken[0]
		if strings.HasPrefix(authTokenStr, "Bearer ") {
			return authTokenStr[7:], TokenJWT, nil
		}
		return "", "", status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	if a.apiKeyEnabled && a.apiKeyValidator != nil {
		apiKeyToken := md[header.HeaderAPIKey]
		if len(apiKeyToken) > 0 {
			return apiKeyToken[0], TokenAPIKey, nil
		}
	}

	return "", "", status.Error(codes.Unauthenticated, "missing authorization header")
}
