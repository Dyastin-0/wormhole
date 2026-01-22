package server

import "github.com/Dyastin-0/wormhole/observer"

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
