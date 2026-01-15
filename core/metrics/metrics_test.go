package metrics

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsReadWriter(t *testing.T) {
	data := []byte("hello world")
	buf := bytes.NewBuffer(data)

	m := New()
	mrw := NewMetricsReadWriter(buf, m)

	readBuf := make([]byte, len(data))
	n, err := mrw.Read(readBuf)
	require.NoError(t, err)
	require.Equal(t, len(data), n)
	require.Equal(t, uint64(n), m.GetIngressBytes())

	writeData := []byte("goodbye")
	n, err = mrw.Write(writeData)
	require.NoError(t, err)
	require.Equal(t, len(writeData), n)
	require.Equal(t, uint64(n), m.GetEgressBytes())
}
