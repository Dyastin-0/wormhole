package proto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDurationLogSerializeDeserialize(t *testing.T) {
	original := NewHTTPDurationLog(uuid.NewString(), uint64(30*time.Millisecond))

	serialized, err := SerializeHTTPDurationLog(original)
	require.NoError(t, err)

	deserialized, err := DeserializeHTTPDurationLog(serialized)
	require.NoError(t, err)

	require.Equal(t, original, deserialized)
}
