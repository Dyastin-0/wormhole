// Package client implents the RPC clients for ../proto
package client

import "google.golang.org/grpc"

func New(addr string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr, opts...)
}
