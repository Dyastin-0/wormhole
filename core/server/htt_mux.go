package server

import (
	"net"
	"net/http"

	"github.com/Dyastin-0/wormhole/stream"
)

type httpMux struct {
	mux *http.ServeMux
}

func newHTTPMux() *httpMux {
	return &httpMux{mux: http.NewServeMux()}
}

func (m *httpMux) Handle(path string, handler http.HandlerFunc) {
	m.mux.HandleFunc(path, handler)
}

func (m *httpMux) Serve(conn net.Conn) error {
	return http.Serve(stream.NewConnListener(conn), m.mux)
}

func (m *httpMux) ServeWithFunc(conn net.Conn, handler http.HandlerFunc) error {
	return http.Serve(stream.NewConnListener(conn), handler)
}
