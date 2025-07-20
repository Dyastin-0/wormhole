// Package header implements custom header matcher for grpc-gateway
package header

import (
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

const (
	HeaderSetCookie     = "set-cookie"
	HeaderAuthorization = "authorization"
	HeaderCookie        = "cookie"
)

type headers map[string]string

var (
	defaultOutgoingHeaders = headers{
		HeaderSetCookie:     "Set-Cookie",
		HeaderAuthorization: "Authorization",
		HeaderCookie:        "Cookie",
	}

	defaultIncomingHeaders = headers{
		"Cookie": "cookie",
	}
)

var (
	DefaultOutgoingHeaderMatcher = OutgoingHeaderMatcher(defaultOutgoingHeaders)
	DefaultIncomingHeaderMatcher = IncomingHeaderMatcher(defaultIncomingHeaders)
)

func IncomingHeaderMatcher(headers headers) runtime.ServeMuxOption {
	return runtime.WithIncomingHeaderMatcher(
		func(key string) (string, bool) {
			if header, ok := headers[key]; ok {
				return header, ok
			}

			return runtime.DefaultHeaderMatcher(key)
		},
	)
}

func OutgoingHeaderMatcher(headers headers) runtime.ServeMuxOption {
	return runtime.WithOutgoingHeaderMatcher(
		func(key string) (string, bool) {
			headers = normalize(headers)

			if header, ok := headers[key]; ok {
				return header, ok
			}

			return runtime.DefaultHeaderMatcher(key)
		},
	)
}

func normalize(h headers) headers {
	out := make(headers)
	for k, v := range h {
		out[strings.ToLower(k)] = v
	}
	return out
}
