package client

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestConnection() (net.Conn, net.Conn) {
	return net.Pipe()
}

func TestClientNewValidConfig(t *testing.T) {
	c, err := New(
		WithAddr("localhost:8080"),
		WithTargetAddr("localhost:3000"),
		WithName("test"),
		WithProto(proto.ProtoHTTP),
	)

	assert.NoError(t, err)
	assert.NotNil(t, c)
}

func TestClientNewInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		opts []OptFunc
	}{
		{
			name: "missing addr",
			opts: []OptFunc{
				WithTargetAddr("localhost:3000"),
				WithName("test"),
				WithProto(proto.ProtoHTTP),
			},
		},
		{
			name: "missing targetAddr",
			opts: []OptFunc{
				WithAddr("localhost:8080"),
				WithName("test"),
				WithProto(proto.ProtoHTTP),
			},
		},
		{
			name: "missing name",
			opts: []OptFunc{
				WithAddr("localhost:8080"),
				WithTargetAddr("localhost:3000"),
				WithProto(proto.ProtoHTTP),
			},
		},
		{
			name: "missing proto",
			opts: []OptFunc{
				WithAddr("localhost:8080"),
				WithTargetAddr("localhost:3000"),
				WithName("test"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.opts...)
			assert.Error(t, err)
			assert.Nil(t, c)
		})
	}
}

func TestClientForwardStreamSuccess(t *testing.T) {
	targetListener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer targetListener.Close()

	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				_, err = c.Write(buf[:n])
				if err != nil {
					t.Logf("Failed to write echo response: %v", err)
				}
			}(conn)
		}
	}()

	c, err := New(
		WithAddr("localhost:8080"),
		WithTargetAddr(targetListener.Addr().String()),
		WithName("test"),
		WithProto(proto.ProtoHTTP),
	)
	require.NoError(t, err)

	streamServer, streamClient := net.Pipe()
	defer streamServer.Close()
	defer streamClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- c.ForwardStream(ctx, streamClient)
	}()

	time.Sleep(100 * time.Millisecond)

	testData := []byte("hello world")
	_, err = streamServer.Write(testData)
	require.NoError(t, err)

	buf := make([]byte, len(testData))
	err = streamServer.SetReadDeadline(time.Now().Add(1 * time.Second))
	require.NoError(t, err)

	n, err := streamServer.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, testData, buf[:n])

	cancel()

	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled && err.Error() != "EOF" {
			t.Logf("ForwardStream ended with: %v", err)
		}
	case <-time.After(1100 * time.Millisecond):
		t.Error("ForwardStream did not return within timeout")
	}
}

func TestForwardStreamFail(t *testing.T) {
	c, err := New(
		WithAddr("localhost:8080"),
		WithTargetAddr("localhost:99999"),
		WithName("test"),
		WithProto(proto.ProtoHTTP),
	)
	require.NoError(t, err)

	streamServer, streamClient := net.Pipe()
	defer streamServer.Close()
	defer streamClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = c.ForwardStream(ctx, streamClient)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to dial tcp target address")
}

func TestClientSendAckSuccess(t *testing.T) {
	c, err := New(
		WithAddr("localhost:8080"),
		WithTargetAddr("localhost:3000"),
		WithName("test"),
		WithProto(proto.ProtoHTTP),
	)
	require.NoError(t, err)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	errChan := make(chan error, 1)
	go func() {
		errChan <- c.sendAck(clientConn)
	}()

	buf := make([]byte, proto.HeaderSize)
	err = serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	require.NoError(t, err)

	_, err = serverConn.Read(buf)
	require.NoError(t, err)

	header, err := proto.DeserializeHeader(buf)
	require.NoError(t, err)
	assert.Equal(t, proto.TypeAck, header.Type)

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Error("sendAck did not return within timeout")
	}
}

func TestSendAckFail(t *testing.T) {
	c, err := New(
		WithAddr("localhost:8080"),
		WithTargetAddr("localhost:3000"),
		WithName("test"),
		WithProto(proto.ProtoHTTP),
	)
	require.NoError(t, err)

	serverConn, clientConn := net.Pipe()
	serverConn.Close()
	defer clientConn.Close()

	err = c.sendAck(clientConn)
	assert.Error(t, err)
}

func TestSendAckClosedConnection(t *testing.T) {
	c, err := New(
		WithAddr("localhost:8080"),
		WithTargetAddr("localhost:3000"),
		WithName("test"),
		WithProto(proto.ProtoHTTP),
	)
	require.NoError(t, err)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	clientConn.Close()

	err = c.sendAck(clientConn)
	assert.Error(t, err)
}
