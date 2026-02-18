package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseSerializeDeserialize(t *testing.T) {
	original := NewResponse(StatusOK, 3600, "test.example.com")

	serialized, err := SerializeResponse(original)
	require.NoError(t, err)

	deserialized, err := DeserializeResponse(serialized)
	require.NoError(t, err)

	require.Equal(t, original, deserialized)
}
