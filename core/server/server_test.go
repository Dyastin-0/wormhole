package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/hashicorp/yamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidConfig(t *testing.T) {
	srv, err := New(
		WithAddr("localhost:8080"),
		WithServeAddr("localhost:8443"),
		WithDomain("test.com"),
	)

	assert.NoError(t, err)
	assert.NotNil(t, srv)
}

func TestNewInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		opts []OptFunc
	}{
		{
			name: "missing addr",
			opts: []OptFunc{
				WithServeAddr("localhost:8443"),
			},
		},
		{
			name: "missing serveAddr",
			opts: []OptFunc{
				WithAddr("localhost:8080"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := New(tt.opts...)
			assert.Error(t, err)
			assert.Nil(t, srv)
		})
	}
}

func TestSendResp(t *testing.T) {
	srv, err := New(
		WithAddr("localhost:8080"),
		WithServeAddr("localhost:8443"),
		WithDomain("test.com"),
	)
	require.NoError(t, err)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	resp := proto.NewResponse(proto.StatusOK, uint64(DefaultTunnelTTL), "domain.com")

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.sendResp(clientConn, resp)
	}()

	buf := make([]byte, proto.HeaderSize)
	_, err = serverConn.Read(buf)
	require.NoError(t, err)

	header, err := proto.DeserializeHeader(buf)
	require.NoError(t, err)
	assert.Equal(t, proto.TypeResponse, header.Type)

	buf = make([]byte, header.Length)
	_, err = serverConn.Read(buf)
	require.NoError(t, err)

	receivedResp, err := proto.DeserializeResponse(buf)
	require.NoError(t, err)
	assert.Equal(t, resp.Status, receivedResp.Status)
	assert.Equal(t, resp.Domain, receivedResp.Domain)

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Error("sendResp did not return within timeout")
	}
}

func TestSendErr(t *testing.T) {
	srv, err := New(
		WithAddr("localhost:8080"),
		WithServeAddr("localhost:8443"),
		WithDomain("test.com"),
	)
	require.NoError(t, err)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	errorMsg := "test error message"

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.sendErr(clientConn, errorMsg)
	}()

	buf := make([]byte, proto.HeaderSize)
	_, err = serverConn.Read(buf)
	require.NoError(t, err)

	header, err := proto.DeserializeHeader(buf)
	require.NoError(t, err)
	assert.Equal(t, proto.TypeError, header.Type)
	assert.Equal(t, uint64(len(errorMsg)), header.Length)

	buf = make([]byte, header.Length)
	_, err = serverConn.Read(buf)
	require.NoError(t, err)

	msg := string(buf)
	require.Equal(t, errorMsg, msg)

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Error("sendErr did not return within timeout")
	}
}

func TestSendEnd(t *testing.T) {
	srv, err := New(
		WithAddr("localhost:8080"),
		WithServeAddr("localhost:8443"),
		WithDomain("test.com"),
	)
	require.NoError(t, err)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	errChan := make(chan error, 1)

	go func() {
		session, errr := yamux.Client(clientConn, nil)
		if errr != nil {
			errChan <- err
		}
		errChan <- srv.sendEnd(session)
	}()

	session, err := yamux.Server(serverConn, nil)
	require.NoError(t, err)
	defer session.Close()

	stream, err := session.Accept()
	require.NoError(t, err)
	defer stream.Close()

	buf := make([]byte, proto.HeaderSize)
	_, err = io.ReadFull(stream, buf)
	require.NoError(t, err)

	header, err := proto.DeserializeHeader(buf)
	require.NoError(t, err)
	assert.Equal(t, proto.TypeEnd, header.Type)

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Error("sendErr did not return within timeout")
	}
}

func createYamuxSessionPair(t *testing.T) (*yamux.Session, *yamux.Session, func()) {
	serverConn, clientConn := net.Pipe()

	serverSession, err := yamux.Server(serverConn, nil)
	require.NoError(t, err)

	clientSession, err := yamux.Client(clientConn, nil)
	require.NoError(t, err)

	cleanup := func() {
		serverSession.Close()
		clientSession.Close()
		serverConn.Close()
		clientConn.Close()
	}

	return serverSession, clientSession, cleanup
}

func TestHandleRequestSuccess(t *testing.T) {
	srv, err := New(
		WithAddr("localhost:8080"),
		WithServeAddr("localhost:8443"),
		WithDomain("test.com"),
	)
	require.NoError(t, err)

	serverSession, clientSession, cleanup := createYamuxSessionPair(t)
	defer cleanup()

	request := proto.NewRequest(proto.ProtoHTTP, "testapp", 0, "")
	serializedRequest, err := proto.SerializeRequest(request)
	require.NoError(t, err)

	header := proto.NewHeader(proto.TypeRequest, uint64(len(serializedRequest)))

	clientStream, err := clientSession.Open()
	require.NoError(t, err)

	errch := make(chan error, 1)

	go func() {
		serverStream, errr := serverSession.Accept()
		if errr != nil {
			errch <- err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		errch <- srv.handleRequest(ctx, serverStream, serverSession, header)
	}()

	_, err = clientStream.Write(serializedRequest)
	require.NoError(t, err)

	err = <-errch
	require.NoError(t, err)
}

func TestHandleRequestNameTaken(t *testing.T) {
	srv, err := New(
		WithAddr("localhost:8080"),
		WithServeAddr("localhost:8443"),
		WithDomain("test.com"),
	)
	require.NoError(t, err)

	srv.tunnels.Set(fmt.Sprintf("testapp.%s", srv.domain), nil)

	serverSession, clientSession, cleanup := createYamuxSessionPair(t)
	defer cleanup()

	request := proto.NewRequest(proto.ProtoHTTP, "testapp", 0, "")
	serializedRequest, err := proto.SerializeRequest(request)
	require.NoError(t, err)

	header := proto.NewHeader(proto.TypeRequest, uint64(len(serializedRequest)))

	clientStream, err := clientSession.Open()
	require.NoError(t, err)

	errch := make(chan error, 1)

	go func() {
		serverStream, errr := serverSession.Accept()
		if errr != nil {
			errch <- err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		errch <- srv.handleRequest(ctx, serverStream, serverSession, header)
	}()

	_, err = clientStream.Write(serializedRequest)
	require.NoError(t, err)

	buf := make([]byte, proto.HeaderSize)
	_, err = clientStream.Read(buf)
	require.NoError(t, err)

	responseHeader, err := proto.DeserializeHeader(buf)
	require.NoError(t, err)
	require.Equal(t, proto.TypeResponse, responseHeader.Type)

	buf = make([]byte, responseHeader.Length)
	_, err = clientStream.Read(buf)
	require.NoError(t, err)

	response, err := proto.DeserializeResponse(buf)
	require.NoError(t, err)
	require.Equal(t, proto.StatusNameTaken, response.Status)

	err = <-errch
	require.Error(t, err)
}

func createInvalidProtocolBytes() []byte {
	name := "testapp"
	buf := make([]byte, 5+len(name))
	buf[0] = 0x03
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(name)))
	copy(buf[5:], []byte(name))
	return buf
}

func TestHandleRequestUnsupportedProto(t *testing.T) {
	srv, err := New(
		WithAddr("localhost:8080"),
		WithServeAddr("localhost:8443"),
		WithDomain("test.com"),
	)
	require.NoError(t, err)

	serverSession, clientSession, cleanup := createYamuxSessionPair(t)
	defer cleanup()

	serializedRequest := createInvalidProtocolBytes()

	header := proto.NewHeader(proto.TypeRequest, uint64(len(serializedRequest)))

	clientStream, err := clientSession.Open()
	require.NoError(t, err)

	errch := make(chan error, 1)

	go func() {
		serverStream, errr := serverSession.Accept()
		if errr != nil {
			errch <- err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		errch <- srv.handleRequest(ctx, serverStream, serverSession, header)
	}()

	_, err = clientStream.Write(serializedRequest)
	require.NoError(t, err)

	buf := make([]byte, proto.HeaderSize)
	_, err = clientStream.Read(buf)
	require.NoError(t, err)

	responseHeader, err := proto.DeserializeHeader(buf)
	require.NoError(t, err)
	require.Equal(t, proto.TypeError, responseHeader.Type)

	err = <-errch
	require.Error(t, err)
}
