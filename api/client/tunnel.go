package client

import (
	"context"

	"github.com/Dyastin-0/wormhole/api/proto/tunnel"
	"google.golang.org/grpc"
)

type TunnelClient struct {
	service tunnel.TunnelServiceClient
}

func NewTunnelClient(addr string, opts ...grpc.DialOption) (*TunnelClient, error) {
	conn, err := New(addr, opts...)
	if err != nil {
		return nil, err
	}

	tc := &TunnelClient{
		service: tunnel.NewTunnelServiceClient(conn),
	}

	return tc, nil
}

func (t *TunnelClient) Create(ctx context.Context, req *tunnel.CreateTunnelRequest) (*tunnel.Tunnel, error) {
	return t.service.CreateTunnel(ctx, req)
}

func (t *TunnelClient) Update(ctx context.Context, req *tunnel.UpdateTunnelRequest) (*tunnel.Tunnel, error) {
	return t.service.UpdateTunnel(ctx, req)
}

func (t *TunnelClient) Get(ctx context.Context, req *tunnel.GetTunnelRequest) (*tunnel.Tunnel, error) {
	return t.service.GetTunnel(ctx, req)
}

func (t *TunnelClient) Delete(ctx context.Context, req *tunnel.DeleteTunnelRequest) (*tunnel.Tunnel, error) {
	return t.service.DeleteTunnel(ctx, req)
}

func (t *TunnelClient) GetMany(ctx context.Context, req *tunnel.GetTunnelsRequest) (*tunnel.GetTunnelsResponse, error) {
	return t.service.GetTunnels(ctx, req)
}
