package core

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Dyastin-0/wormhole/core/client"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/server"
	"github.com/stretchr/testify/require"
)

func startTestHTTPServer(addr string) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from target server!")
	})
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Test response: %s", r.URL.Query().Get("msg"))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		server.ListenAndServe()
	}()

	return server, nil
}

func NewTestCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: "*.app.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"*.app.com", "app.com"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}

func TestRequestResponse(t *testing.T) {
	targetServer, err := startTestHTTPServer("localhost:8591")
	require.NoError(t, err)
	defer targetServer.Close()

	time.Sleep(100 * time.Millisecond)

	srv, err := server.New(
		server.WithAddr("localhost:0"),
		server.WithServeAddr("localhost:0"),
		server.WithDomain("app.com"),
	)
	require.NoError(t, err)

	controlListener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer controlListener.Close()

	cert, err := NewTestCert()
	require.NoError(t, err)

	sTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
	}

	tunnelListener, err := tls.Listen("tcp", "localhost:0", sTLSConfig)
	require.NoError(t, err)
	defer tunnelListener.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	sErrch := make(chan error, 2)
	go func() {
		sErrch <- srv.RunWithListener(ctx, controlListener)
	}()
	go func() {
		sErrch <- srv.RunTunnelerWithListener(ctx, tunnelListener)
	}()

	time.Sleep(100 * time.Millisecond)

	c, err := client.New(
		client.WithAddr(controlListener.Addr().String()),
		client.WithName("testapp"),
		client.WithProto(proto.ProtoHTTP),
		client.WithTargetAddr("localhost:8591"),
	)
	require.NoError(t, err)

	cErrch := make(chan error, 1)
	go func() {
		cErrch <- c.RunWithTCP(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "testapp.app.com",
	}

	tunnelConn, err := tls.Dial("tcp", tunnelListener.Addr().String(), tlsConfig)
	require.NoError(t, err)

	request := "GET /test?msg=hello HTTP/1.1\r\nHost: testapp.app.com\r\n\r\n"
	_, err = tunnelConn.Write([]byte(request))
	require.NoError(t, err)

	response := make([]byte, 1024)
	n, err := tunnelConn.Read(response)
	require.NoError(t, err)
	require.NotEqual(t, 0, n)
	responseStr := string(response[:n])
	require.Contains(t, responseStr, "Test response: hello")
	tunnelConn.Close()

	cancel()

	select {
	case err := <-sErrch:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
	}

	select {
	case err := <-cErrch:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
	}

	time.Sleep(50 * time.Millisecond)
}

func TestRequestResponseNameTaken(t *testing.T) {
	srv, err := server.New(
		server.WithAddr("localhost:0"),
		server.WithServeAddr("localhost:0"),
		server.WithDomain("app.com"),
	)
	require.NoError(t, err)

	controlListener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer controlListener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sErrch := make(chan error, 2)
	go func() {
		sErrch <- srv.RunWithListener(ctx, controlListener)
	}()

	time.Sleep(100 * time.Millisecond)

	c1, err := client.New(
		client.WithAddr(controlListener.Addr().String()),
		client.WithName("testapp"),
		client.WithProto(proto.ProtoHTTP),
		client.WithTargetAddr("localhost:9090"),
	)
	require.NoError(t, err)

	c1Errch := make(chan error, 1)
	go func() {
		c1Errch <- c1.RunWithTCP(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	c2, err := client.New(
		client.WithAddr(controlListener.Addr().String()),
		client.WithName("testapp"),
		client.WithProto(proto.ProtoHTTP),
		client.WithTargetAddr("localhost:9090"),
	)
	require.NoError(t, err)

	c2Errch := make(chan error, 1)
	go func() {
		c2Errch <- c2.RunWithTCP(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-c2Errch:
		require.Error(t, err)
		require.Contains(t, err.Error(), "name taken")
	case <-time.After(1 * time.Second):
		t.Fatal("expected second client to fail with name taken error")
	}

	cancel()

	select {
	case err := <-sErrch:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
	}

	select {
	case err := <-c1Errch:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
	}
}
