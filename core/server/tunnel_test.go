package server

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Dyastin-0/wormhole/core/metrics"
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/hashicorp/yamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromSuccess(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	streamServerConn, streamClientConn := net.Pipe()
	defer streamServerConn.Close()
	defer streamClientConn.Close()

	go func() {
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
		assert.Equal(t, proto.TypeAccess, header.Type)

		ackHeader := proto.NewHeader(proto.TypeAck, 0)
		serializedAck, err := proto.SerializeHeader(ackHeader)
		require.NoError(t, err)
		_, err = stream.Write(serializedAck)
		require.NoError(t, err)

		go io.Copy(stream, streamServerConn)
		io.Copy(streamServerConn, stream)
	}()

	go func() {
		time.Sleep(100 * time.Millisecond)
		_, err := streamClientConn.Write([]byte("test data"))
		if err != nil {
			t.Logf("Failed to write test data: %v", err)
		}

		streamClientConn.Close()
	}()

	session, err := yamux.Client(clientConn, nil)
	require.NoError(t, err)
	defer session.Close()

	tunnel := &Tunnel{
		session: session,
		proto:   proto.ProtoHTTP,
		metrics: metrics.NewMetrics(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = tunnel.From(ctx, streamClientConn)
	if errors.Is(err, context.DeadlineExceeded) {
		return
	}
	require.Error(t, err)
}

func TestFromHandshakeFail(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	streamServerConn, streamClientConn := net.Pipe()
	defer streamServerConn.Close()
	defer streamClientConn.Close()

	go func() {
		session, err := yamux.Server(serverConn, nil)
		require.NoError(t, err)
		defer session.Close()

		stream, err := session.Accept()
		require.NoError(t, err)
		defer stream.Close()

		buf := make([]byte, proto.HeaderSize)
		_, err = io.ReadFull(stream, buf)
		require.NoError(t, err)

		wrongHeader := proto.NewHeader(proto.TypeRequest, 0)
		serializedWrongHeader, err := proto.SerializeHeader(wrongHeader)
		require.NoError(t, err)
		_, err = stream.Write(serializedWrongHeader)
		require.NoError(t, err)
	}()

	session, err := yamux.Client(clientConn, nil)
	require.NoError(t, err)
	defer session.Close()

	tunnel := &Tunnel{
		session:   session,
		proto:     proto.ProtoHTTP,
		dnsRecord: &dnsmanager.DNSRecord{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = tunnel.From(ctx, streamClientConn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected header type")
}

func TestFromSessionOpenFail(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	serverConn.Close()
	clientConn.Close()

	session, err := yamux.Client(clientConn, nil)
	require.NoError(t, err)
	session.Close()

	streamServerConn, streamClientConn := net.Pipe()
	defer streamServerConn.Close()
	defer streamClientConn.Close()

	tunnel := &Tunnel{
		session: session,
		proto:   proto.ProtoHTTP,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = tunnel.From(ctx, streamClientConn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open yamux session")
}

func TestFromWriteHeaderFail(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	streamServerConn, streamClientConn := net.Pipe()
	defer streamServerConn.Close()
	defer streamClientConn.Close()

	go func() {
		session, err := yamux.Server(serverConn, nil)
		require.NoError(t, err)
		defer session.Close()

		stream, err := session.Accept()
		require.NoError(t, err)
		stream.Close()
	}()

	session, err := yamux.Client(clientConn, nil)
	require.NoError(t, err)
	defer session.Close()

	tunnel := &Tunnel{
		session: session,
		proto:   proto.ProtoHTTP,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = tunnel.From(ctx, streamClientConn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "EOF")
}
