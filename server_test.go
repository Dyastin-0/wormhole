package wormhole

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/hashicorp/yamux"
)

func TestNew(t *testing.T) {
	w := New(":8888", ":9999")

	if w.addr != ":8888" {
		t.Errorf("addr mismatch: got %s, want %s", w.addr, ":8888")
	}

	if w.httpAddr != ":9999" {
		t.Errorf("httpAddr mismatch: got %s, want %s", w.httpAddr, ":9999")
	}
}

func TestStart(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		httpAddr string
		wantErr  bool
	}{
		{
			name:     "invalid address",
			addr:     "8000",
			httpAddr: "2000",
			wantErr:  true,
		},
		{
			name:     "valid address",
			addr:     ":8001",
			httpAddr: ":8002",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(tt.addr, tt.httpAddr)
			dbStore := store.New(nil)
			dnsManager := &dnsmanager.Manager{
				API: newMockDNSAPI("dyastin.tech", "127.0.0.1"),
			}

			w.Store = dbStore
			w.DNSManager = dnsManager

			errChan := make(chan error, 1)

			go func() {
				errChan <- w.Start(context.Background())
			}()

			select {
			case err := <-errChan:
				if (err != nil) != tt.wantErr {
					t.Errorf("Start() error = %v, wantErr = %v", err, tt.wantErr)
				}
			case <-time.After(200 * time.Millisecond):
				if tt.wantErr {
					t.Error("expected an error but Start() did not return in time")
				} else {
					w.Stop()
				}
			}
		})
	}
}

func TestStop(t *testing.T) {
	w := New(":8083", ":8010")

	store := store.New(nil)
	dnsManager := &dnsmanager.Manager{
		API: newMockDNSAPI("dyastin.tech", "127.0.0.1"),
	}

	w.Store = store
	w.DNSManager = dnsManager

	done := make(chan error, 1)

	go func() {
		err := w.Start(context.Background())
		done <- err
	}()

	time.Sleep(100 * time.Millisecond)

	w.Stop()

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Start() returned error after Stop(): %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start() did not return within 1 second after Stop()")
	}
}

func TestHandshake(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	w := &Wormhole{}

	go func() {
		msg := message{
			TunnelName:  "test",
			TunnelProto: "http",
		}

		enc := json.NewEncoder(client)
		dec := json.NewDecoder(client)

		if err := enc.Encode(msg); err != nil {
			t.Error(err)
			return
		}

		var resp message

		if err := dec.Decode(&resp); err != nil {
			t.Error(err)
			return
		}

		if resp.Status != 0 {
			t.Errorf("expected resp.status=0, got: %v, err: %s", resp.Status, resp.Err)
		}
	}()

	msg, _, err := w.handshake(server)
	if err != nil {
		t.Fatal(err)
	}

	if msg.TunnelName != "test" {
		t.Errorf("expected msg.ID=test, got %s", msg.TunnelName)
	}

	if msg.TunnelProto != "http" {
		t.Errorf("expected msg.Proto=http, got %s", msg.TunnelProto)
	}
}

func TestHTTP(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	defer ln.Close()

	w := &Wormhole{
		DNSManager: &dnsmanager.Manager{
			API: newMockDNSAPI("wormhole.dyastin.tech", "127.0.0.1"),
		},
	}

	go func() {
		conn, errr := ln.Accept()
		if errr != nil {
			t.Error(err)
			return
		}

		_ = w.handleConn(conn)
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	session, err := yamux.Client(conn, nil)
	if err != nil {
		t.Fatal(err)
	}

	stream, err := session.Open()
	if err != nil {
		t.Fatal(err)
	}

	msg := &message{
		TunnelName:  "foo",
		TunnelProto: "http",
	}

	enc := json.NewEncoder(stream)
	dec := json.NewDecoder(stream)

	if err := enc.Encode(msg); err != nil {
		t.Fatal(err)
	}

	var resp message

	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Status != 0 {
		t.Fatalf("expected resp.status=%d, but got %d", 0, resp.Status)
	}

	time.Sleep(200 * time.Millisecond)

	domain := fmt.Sprintf("%s.%s", msg.TunnelName, w.DNSManager.API.BaseDNS())

	_, exists := w.tunnels.Load(domain)
	if !exists {
		t.Errorf("expected %s to be registered but was not found", domain)
	}

	session.Close()
	time.Sleep(100 * time.Millisecond)

	_, exists = w.tunnels.Load(domain)

	if exists {
		t.Errorf("expected %s to be deleted but was found", domain)
	}
}

func TestTunneler(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	w := &Wormhole{
		tunnelHTTPRequest: tunnelHTTPRequest,
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	go func() {
		bufr := bufio.NewReader(server)

		_, err := http.ReadRequest(bufr)
		if err != nil {
			t.Error(err)
			return
		}

		resp := &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("Hello from wormhole")),
		}
		resp.Header.Set("X-Test", "true")

		err = resp.Write(server)
		if err != nil {
			t.Error(err)
			return
		}

		server.Close()
	}()

	err := w.tunnelHTTPRequest(client, rr, req)
	if err != nil {
		t.Error(err)
	}

	res := rr.Result()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != 200 {
		t.Errorf("expected res.StatusCode=200, got %d", res.StatusCode)
	}

	if res.Header.Get("X-Test") != "true" {
		t.Errorf("expected res.Header X-Test=true, got %v", res.Header.Get("X-Test"))
	}

	if string(body) != "Hello from wormhole" {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestWormhole_HTTP(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	mock := &mockSession{conn: clientConn}

	dnsManager := &dnsmanager.Manager{
		API: newMockDNSAPI("dyastin.tech", "127.0.0.1"),
	}

	w := &Wormhole{
		DNSManager: dnsManager,
		tunnelHTTPRequest: func(stream net.Conn, wr http.ResponseWriter, r *http.Request) error {
			wr.WriteHeader(http.StatusTeapot)
			wr.Write([]byte("tunneled!"))
			return nil
		},
	}

	newTunnel := &tunnel{
		proto:   "http",
		session: mock,
	}

	w.tunnels.Store("test.dyastin.tech", newTunnel)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Host", "test.dyastin.tech")

	rr := httptest.NewRecorder()

	w.HTTP(rr, req)

	res := rr.Result()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusTeapot {
		t.Errorf("expected 418, got %d", res.StatusCode)
	}
	if string(body) != "tunneled!" {
		t.Errorf("unexpected body: %q", body)
	}
}
