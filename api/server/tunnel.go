package server

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/api/interceptor"
	tunnelpb "github.com/Dyastin-0/wormhole/api/proto/tunnel"
	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TunnelServer struct {
	tunnelpb.UnimplementedTunnelServiceServer
	store *store.Store
}

func NewTunnelServer(store *store.Store) *TunnelServer {
	return &TunnelServer{
		store: store,
	}
}

func (t *TunnelServer) CreateTunnel(ctx context.Context, req *tunnelpb.CreateTunnelRequest) (*tunnelpb.Tunnel, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgName)
	}

	if req.Domain == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgDomain)
	}

	if req.Protocol == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgProtocol)
	}

	if req.Target == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgTarget)
	}

	if req.Ipv4 == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgIPV4)
	}

	userID := ctx.Value(interceptor.CtxPayload("user_id"))
	if userID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "%s: %s", ErrMissingCtxPayload, "user_id")
	}

	id := uuid.New().String()
	param := &db.CreateTunnelParams{
		ID:       id,
		UserID:   userID.(string),
		Name:     req.Name,
		Domain:   req.Domain,
		Protocol: req.Protocol,
		Target:   req.Target,
		Ipv4:     req.Ipv4,
	}

	res, err := t.store.Tunnel.Create(ctx, param)
	if err != nil {
		switch {
		case isUniqueConstraintOn(err, "tunnels.domain"):
			return nil, status.Errorf(codes.AlreadyExists, "%s: %s", ErrFailedToCreateTunnel, ErrTunnelDomainAlreadyUsed)

		case isUniqueConstraintOn(err, "tunnels.name"):
			return nil, status.Errorf(codes.AlreadyExists, "%s: %s", ErrFailedToCreateTunnel, ErrTunnelNameAlreadyUsed)

		default:
			return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToCreateTunnel, err)
		}
	}

	return &tunnelpb.Tunnel{
		Id:        res.ID,
		UserId:    res.UserID,
		Name:      res.Name,
		Domain:    res.Domain,
		Protocol:  res.Protocol,
		Target:    res.Target,
		Ipv4:      res.Ipv4,
		CreatedAt: res.CreatedAt.Time.String(),
	}, nil
}

func (t *TunnelServer) GetTunnel(ctx context.Context, req *tunnelpb.GetTunnelRequest) (*tunnelpb.Tunnel, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgID)
	}

	param := &db.GetTunnelParams{
		ID:     req.Id,
		UserID: req.UserId,
	}

	res, err := t.store.Tunnel.Get(ctx, param)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, ErrTunnelNotFound)
		}
		return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToGetTunnel, err)
	}

	return &tunnelpb.Tunnel{
		Id:        res.ID,
		UserId:    res.UserID,
		Domain:    res.Domain,
		Name:      res.Name,
		Protocol:  res.Protocol,
		Target:    res.Target,
		Ipv4:      res.Ipv4,
		CreatedAt: res.CreatedAt.Time.String(),
		UpdatedAt: res.UpdatedAt.Time.String(),
	}, nil
}

func (t *TunnelServer) UpdateTunnel(ctx context.Context, req *tunnelpb.UpdateTunnelRequest) (*tunnelpb.Tunnel, error) {
	if req.Id == "" {
		return nil, status.Error(codes.Internal, ErrMissingArgID)
	}

	param := &db.UpdateTunnelParams{
		ID:       req.Id,
		Domain:   toNullString(req.Domain),
		Name:     toNullString(req.Name),
		Protocol: toNullString(req.Protocol),
		Target:   toNullString(req.Target),
		Ipv4:     toNullString(req.Ipv4),
	}

	res, err := t.store.Tunnel.Update(ctx, param)
	if err != nil {
		switch {
		case isUniqueConstraintOn(err, "tunnels.domain"):
			return nil, status.Errorf(codes.AlreadyExists, "%s: %s", ErrFailedToUpdateTunnel, ErrEmailAlreadyUsed)

		case isUniqueConstraintOn(err, "tunnels.name"):
			return nil, status.Errorf(codes.AlreadyExists, "%s: %s", ErrFailedToUpdateTunnel, ErrTunnelNameAlreadyUsed)

		default:
			return nil, status.Errorf(codes.Internal, "%s: %v", ErrFailedToUpdateTunnel, err)
		}
	}

	return &tunnelpb.Tunnel{
		Id:        res.ID,
		UserId:    res.UserID,
		Domain:    res.Domain,
		Name:      res.Name,
		Protocol:  res.Protocol,
		Target:    res.Target,
		Ipv4:      res.Ipv4,
		CreatedAt: res.CreatedAt.Time.String(),
		UpdatedAt: res.UpdatedAt.Time.String(),
	}, nil
}

func (t *TunnelServer) DeleteTunnel(ctx context.Context, req *tunnelpb.DeleteTunnelRequest) (*tunnelpb.Tunnel, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgID)
	}

	userID := ctx.Value(interceptor.CtxPayload("user_id"))
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, ErrMissingArgUserID)
	}

	param := &db.DeleteTunnelParams{
		ID:     req.Id,
		UserID: userID.(string),
	}

	res, err := t.store.Tunnel.Delete(ctx, param)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrFailedToDeleteTunnel)
	}

	return &tunnelpb.Tunnel{
		Id:        res.ID,
		UserId:    res.UserID,
		Domain:    res.Domain,
		Name:      res.Name,
		Protocol:  res.Protocol,
		Target:    res.Target,
		Ipv4:      res.Ipv4,
		CreatedAt: res.CreatedAt.Time.String(),
		UpdatedAt: res.UpdatedAt.Time.String(),
	}, nil
}
