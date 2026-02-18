package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderSerializeDeserialize(t *testing.T) {
	original := NewHeader(TypeRequest, 1024)

	serialized, err := SerializeHeader(original)
	require.NoError(t, err)

	deserialized, err := DeserializeHeader(serialized)
	require.NoError(t, err)

	require.Equal(t, original, deserialized)
}

func TestValidateHeaderInvalidVersion(t *testing.T) {
	header := &Header{
		Version:  0x10,
		Type:     TypeRequest,
		Length:   100,
		Reserved: 0,
	}

	_, err := SerializeHeader(header)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version")
}

func TestFlag(t *testing.T) {
	header := NewHeader(TypeEnd, 0)
	header.SetFlag(FlagMetrics)
	require.True(t, header.HasFlag(FlagMetrics))
	header.ClearFlag(FlagMetrics)
	require.False(t, header.HasFlag(FlagMetrics))
}

func TestIsValidType(t *testing.T) {
	assert.True(t, IsValidType(TypeRequest))
	assert.True(t, IsValidType(TypeResponse))
	assert.True(t, IsValidType(TypeError))
	assert.False(t, IsValidType(0xEE))
}
