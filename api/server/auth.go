package server

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/api/header"
	authpb "github.com/Dyastin-0/wormhole/api/proto/auth"
	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/Dyastin-0/wormhole/hash"
	"github.com/Dyastin-0/wormhole/token"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	store  *store.Store
	issuer *token.Issuer
	hash   hash.Hash
}

func NewAuthServer(store *store.Store, tokenIssuer *token.Issuer) *AuthServer {
	if tokenIssuer == nil {
		tokenIssuer = token.DefaultIssuer()
	}

	return &AuthServer{
		store:  store,
		issuer: tokenIssuer,
		hash:   defaultHash,
	}
}

func (a *AuthServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
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

	byteHashedPassword, err := a.hash.Generate([]byte(req.Password))
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

	res, err := a.store.User.Create(ctx, reqq)
	if err != nil {
		switch {
		case isUniqueConstraintOn(err, "users.email"):
			return nil, status.Errorf(codes.AlreadyExists, "%s: %s", ErrFailedToCreateUser, ErrEmailAlreadyUsed)

		case isUniqueConstraintOn(err, "users.username"):
			return nil, status.Errorf(codes.AlreadyExists, "%s: %s", ErrFailedToCreateUser, ErrUsernameAlreadyUsed)

		default:
			return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToCreateUser, err)
		}
	}

	accessToken, err := a.issuer.NewAccessToken(res.ID, res.Email, token.RoleUser, token.DefaultAccessTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGenerateAccessToken)
	}

	refreshToken, err := a.issuer.NewRefreshToken(res.ID, res.Email, token.RoleUser, token.DefaultRefreshTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGenerateRefreshToken)
	}

	param := &db.CreateRefreshTokenParams{
		ID:     refreshToken,
		UserID: res.ID,
	}

	err = a.store.Auth.Create(ctx, param)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrInternal)
	}

	setCookie(
		ctx,
		token.KeyRefreshToken,
		accessToken,
		time.Now().Add(token.DefaultRefreshTTL),
	)

	return &authpb.AuthResponse{
		AccessToken: accessToken,
		User: &authpb.User{
			Id:       res.ID,
			Email:    res.Email,
			Username: res.Username,
			Name:     res.Name,
		},
	}, nil
}

func (a *AuthServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgEmail)
	}

	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgPassword)
	}

	user, err := a.store.User.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, ErrUserNotFound)
		}
		return nil, status.Errorf(codes.Internal, ErrFailedToGetUser)
	}

	err = a.hash.Compare([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, ErrInvalidPassword)
	}

	accessToken, err := a.issuer.NewAccessToken(user.ID, user.Email, token.RoleUser, token.DefaultAccessTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGenerateAccessToken)
	}

	refreshToken, err := a.issuer.NewRefreshToken(user.ID, user.Email, token.RoleUser, token.DefaultRefreshTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGenerateRefreshToken)
	}

	param := &db.CreateRefreshTokenParams{
		ID:     refreshToken,
		UserID: user.ID,
	}

	err = a.store.Auth.Create(ctx, param)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrInternal)
	}

	setCookie(
		ctx,
		token.KeyRefreshToken,
		refreshToken,
		time.Now().Add(token.DefaultRefreshTTL),
	)

	return &authpb.AuthResponse{
		AccessToken: accessToken,
		User: &authpb.User{
			Email: user.Email,
			Id:    user.ID,
			Name:  user.Name,
		},
	}, nil
}

func (a *AuthServer) Refresh(ctx context.Context, req *authpb.RefreshRequest) (*authpb.AuthResponse, error) {
	cookies, err := getCookies(ctx)
	if err != nil {
		return nil, err
	}

	refreshToken := getCookie("refresh_token", cookies)
	if refreshToken == "" {
		return nil, status.Error(codes.Unauthenticated, ErrMissingRefreshToken)
	}

	payload, err := a.issuer.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, ErrInvalidRefreshToken)
	}

	id := (*payload)[token.PayloadID].(string)
	role := (*payload)[token.PayloadRole].(string)

	getTokenParam := &db.GetRefreshTokenParams{
		ID:     refreshToken,
		UserID: id,
	}

	_, err = a.store.Auth.Get(ctx, getTokenParam)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			setCookie(
				ctx,
				token.KeyRefreshToken,
				"",
				time.Unix(0, 0),
			)
			return nil, status.Error(codes.NotFound, ErrRefreshTokenNotFound)
		}
		return nil, status.Error(codes.Internal, ErrFailedToGetRefreshToken)
	}

	user, err := a.store.User.Get(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGetUser)
	}

	newAccessToken, err := a.issuer.NewAccessToken(user.ID, user.Email, role, token.DefaultAccessTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGenerateAccessToken)
	}

	newRefreshToken, err := a.issuer.NewRefreshToken(user.ID, user.Email, role, token.DefaultRefreshTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToGenerateRefreshToken)
	}

	createTokenParam := &db.CreateRefreshTokenParams{
		ID:     newRefreshToken,
		UserID: user.ID,
	}

	err = a.store.Auth.Create(ctx, createTokenParam)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s: %s", ErrFailedToRefreshToken, err)
	}

	deleteTokenParam := &db.DeleteRefreshTokenParams{
		ID:     refreshToken,
		UserID: id,
	}

	err = a.store.Auth.Delete(ctx, deleteTokenParam)
	if err != nil {
		log.Printf("failed to delete old refresh token: %v", err)
	}

	setCookie(
		ctx,
		token.KeyRefreshToken,
		newRefreshToken,
		time.Now().Add(token.DefaultRefreshTTL),
	)

	return &authpb.AuthResponse{
		AccessToken: newAccessToken,
		User: &authpb.User{
			Email: user.Email,
			Id:    user.ID,
			Name:  user.Name,
		},
	}, nil
}

func (a *AuthServer) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	cookies, err := getCookies(ctx)
	if err != nil {
		return nil, err
	}

	refreshToken := getCookie("refresh_token", cookies)
	if refreshToken == "" {
		return nil, status.Error(codes.Unauthenticated, ErrMissingRefreshToken)
	}

	payload, err := a.issuer.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrInvalidRefreshToken)
	}

	id := (*payload)[token.PayloadID].(string)
	deleteTokenParam := &db.DeleteRefreshTokenParams{
		ID:     refreshToken,
		UserID: id,
	}

	err = a.store.Auth.Delete(ctx, deleteTokenParam)
	if err != nil {
		return nil, err
	}

	setCookie(
		ctx,
		token.KeyRefreshToken,
		"",
		time.Unix(0, 0),
	)

	return &authpb.LogoutResponse{}, nil
}

func getCookies(ctx context.Context) ([]string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, ErrMissingMetadata)
	}

	cookies := md.Get(header.HeaderCookie)

	if len(cookies) == 0 {
		return nil, status.Error(codes.Unauthenticated, ErrMissingCookieHeader)
	}

	return cookies, nil
}

func setCookie(ctx context.Context, name, value string, expires time.Time) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  expires,
		SameSite: http.SameSiteLaxMode,
	}

	header := metadata.Pairs(header.HeaderSetCookie, cookie.String())
	grpc.SetHeader(ctx, header)
}

func getCookie(key string, cookies []string) string {
	for _, raw := range cookies {
		req := &http.Request{Header: http.Header{"Cookie": []string{raw}}}
		for _, c := range req.Cookies() {
			if c.Name == key {
				return c.Value
			}
		}
	}
	return ""
}
