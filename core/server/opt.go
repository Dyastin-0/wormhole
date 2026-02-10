package server

import (
	"crypto/tls"

	"github.com/Dyastin-0/wormhole/observer"
	"go.opentelemetry.io/otel/trace"
)

type OptFunc func(*Server)

func WithAddr(addr string) OptFunc {
	return func(s *Server) {
		s.addr = addr
	}
}

func WithServeAddr(addr string) OptFunc {
	return func(s *Server) {
		s.serveAddr = addr
	}
}

func WithDomain(domain string) OptFunc {
	return func(s *Server) {
		s.domain = domain
	}
}

func WithAPIKeyIssuer(apiKeyIssuer *APIKeyIssuer) OptFunc {
	return func(s *Server) {
		s.apiKeyIssuer = apiKeyIssuer
	}
}

func WithObserver(observer observer.Observer) OptFunc {
	return func(s *Server) {
		s.observer = observer
	}
}

func WithTracer(tracer trace.Tracer) OptFunc {
	return func(s *Server) {
		s.tracer = tracer
	}
}

func WithAllowTCP(s *Server) {
	s.allowTCP = true
}

func WithTLSConfig(config *tls.Config) OptFunc {
	return func(s *Server) {
		s.tlsConfig = config
	}
}
