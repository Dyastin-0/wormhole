package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsSerializeDeserialize(t *testing.T) {
	original := NewMetrics(1024, 1024, 1024, 1024, 1024, 54)
	serialized, err := SerializeMetrics(original)
	require.NoError(t, err)

	deserialized, err := DeserializeMetrics(serialized)
	require.NoError(t, err)

	require.Equal(t, original, deserialized)
}
