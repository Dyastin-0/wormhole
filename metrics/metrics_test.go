package metrics

import (
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsReadWriteCloser(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	m := New()
	mrw := NewMetricsReadWriteCloser(a, m)

	data := []byte("hello world")

	go func() {
		_, err := b.Write(data)
		require.NoError(t, err)
	}()

	readBuf := make([]byte, len(data))
	n, err := mrw.Read(readBuf)
	require.NoError(t, err)
	require.Equal(t, len(data), n)
	require.Equal(t, uint64(n), m.GetIngressBytes())
	require.Equal(t, data, readBuf)

	writeData := []byte("goodbye")

	go func() {
		buf := make([]byte, len(writeData))
		_, err = io.ReadFull(b, buf)
		require.NoError(t, err)
		require.Equal(t, writeData, buf)
	}()

	n, err = mrw.Write(writeData)
	require.NoError(t, err)
	require.Equal(t, len(writeData), n)
	require.Equal(t, uint64(n), m.GetEgressBytes())
}
