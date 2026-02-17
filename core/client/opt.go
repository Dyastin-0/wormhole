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

func WithURL(url string) OptFunc {
	return func(c *Client) {
		c.url = url
	}
}

func WithProto(proto uint8) OptFunc {
	return func(c *Client) {
		c.proto = proto
	}
}

func WithProtoTLS(c *Client) {
	c.proto = proto.ProtoTLS
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

func WithBasicAuth(username, password string) OptFunc {
	return func(c *Client) {
		c.authType = proto.AuthTypeBasic
		c.authUsername = username
		c.authPassword = password
	}
}

func WithBearerAuth(token string) OptFunc {
	return func(c *Client) {
		c.authType = proto.AuthTypeBearer
		c.authToken = token
	}
}

func WithNoAuth(c *Client) {
	c.authType = proto.AuthTypeNone
}

func WithAllowHTTP(allowHTTP bool) OptFunc {
	return func(c *Client) {
		c.allowHTTP = allowHTTP
	}
}

func WithHTTPLog(logHTTP bool) OptFunc {
	return func(c *Client) {
		c.LogHTTP = logHTTP
	}
}

func WithAllowTLSPassthrough(c *Client) {
	c.allowTLSPassthrough = true
}
