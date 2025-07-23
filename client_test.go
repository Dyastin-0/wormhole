package wormhole

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

func TestNewClient(t *testing.T) {
	c := NewClient("", "", "test", "localhost:9999", "localhost:8000", "http")

	if c.name != "test" || c.wormholeAddr != "localhost:9999" || c.targetAddr != "localhost:8000" || c.proto != "http" {
		t.Errorf("mismatch in client fields")
	}
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	keyDER, _ := x509.MarshalECPrivateKey(priv)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func TestClientStart(t *testing.T) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		panic(err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
	}

	ln, err := tls.Listen("tcp", ":0", tlsConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Error(err)
			return
		}

		sess, _ := yamux.Server(conn, nil)
		stream, _ := sess.Accept()

		dec := json.NewDecoder(stream)
		enc := json.NewEncoder(stream)

		var msg message
		_ = dec.Decode(&msg)

		msg.Status = 0

		_ = enc.Encode(&msg)

		time.Sleep(200 * time.Millisecond)

		sess.Close()
	}()

	c := NewClient("", "", "test", ln.Addr().String(), ":0", "http")
	errCh := make(chan error, 1)
	go func() { errCh <- c.Start(context.Background(), InsecureTLSConfig()) }()
	time.Sleep(300 * time.Millisecond)
	c.Stop()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not return in time")
	}
}

func TestClientStopWithoutStart(t *testing.T) {
	c := NewClient("", "", "id", "", "", "http")
	c.Stop()
}

func TestClientHandshake(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	c := NewClient("", "", "abc", "", "", "http")

	go func() {
		dec := json.NewDecoder(client)
		enc := json.NewEncoder(client)
		var msg message
		_ = dec.Decode(&msg)
		msg.Status = 0
		_ = enc.Encode(&msg)
	}()

	msg, err := c.handshake(server)
	if err != nil {
		t.Fatal(err)
	}
	if msg.TunnelName != "abc" || msg.TunnelProto != "http" {
		t.Errorf("invalid handshake response: %+v", msg)
	}
}

func TestClientHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "ok")
		w.WriteHeader(200)
		w.Write([]byte("data"))
	}))
	defer ts.Close()

	addr := strings.TrimPrefix(ts.URL, "http://")
	c := NewClient("", "", "id", "", addr, "http")

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		req, _ := http.NewRequest("GET", "/", nil)
		_ = req.Write(client)

		resp, err := http.ReadResponse(bufio.NewReader(client), req)
		if err != nil {
			t.Errorf("failed to read response: %v", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "data" {
			t.Errorf("unexpected body: %s", body)
		}
		if resp.Header.Get("X-Test") != "ok" {
			t.Errorf("expected res.Header X-Test=ok, but got %v", resp.Header.Get("X-Test"))
		}
	}()

	if err := c.http(server); err != nil {
		t.Fatal(err)
	}
}

func TestClientHandleConnUnsupported(t *testing.T) {
	c := NewClient("", "", "name", "", "", "invalid")
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	if err := c.handleConn(server); !errors.Is(err, ErrUnsupportedProtocol) {
		t.Errorf("expected unsupported protocol error, got: %v", err)
	}
}
