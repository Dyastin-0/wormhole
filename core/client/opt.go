package client

import (
	"github.com/Dyastin-0/wormhole/core/proto"
)

type OptFunc func(*Client)

func WithAddr(addr string) OptFunc {
	return func(c *Client) {
		c.addr = addr
	}
}

func WithTargetAddr(addr string) OptFunc {
	return func(c *Client) {
		c.targetAddr = addr
	}
}

func WithName(name string) OptFunc {
	return func(c *Client) {
		c.name = name
	}
}

func WithProto(proto uint8) OptFunc {
	return func(c *Client) {
		c.proto = proto
	}
}

func WithProtoTCP(c *Client) {
	c.proto = proto.ProtoTCP
}

func WithProtoHTTP(c *Client) {
	c.proto = proto.ProtoHTTP
}

func WithMetrics(metrics bool) OptFunc {
	return func(c *Client) {
		c.metrics = metrics
	}
}

func WithAPIKey(apiKey string) OptFunc {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

func WithTTL(ttl uint64) OptFunc {
	return func(c *Client) {
		c.ttl = ttl
	}
}
