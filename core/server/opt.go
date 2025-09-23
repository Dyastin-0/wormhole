package server

import "github.com/Dyastin-0/wormhole/dnsmanager"

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

func WithDNSManager(dnsManager dnsmanager.DNSManager) OptFunc {
	return func(s *Server) {
		s.dnsManager = dnsManager
	}
}
