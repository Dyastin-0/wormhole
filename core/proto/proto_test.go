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

	assert.Equal(t, original.Version, deserialized.Version)
	assert.Equal(t, original.Type, deserialized.Type)
	assert.Equal(t, original.Length, deserialized.Length)
	assert.Equal(t, original.Reserved, deserialized.Reserved)
}

func TestRequestSerializeDeserialize(t *testing.T) {
	original := NewRequest(ProtoHTTP, "test-subdomain")

	serialized, err := SerializeRequest(original)
	require.NoError(t, err)

	deserialized, err := DeserializeRequest(serialized)
	require.NoError(t, err)

	assert.Equal(t, original.Proto, deserialized.Proto)
	assert.Equal(t, original.Name, deserialized.Name)
	assert.Equal(t, original.NameLength, deserialized.NameLength)
}

func TestResponseSerializeDeserialize(t *testing.T) {
	original := NewResponse(StatusOK, 3600, "test.example.com")

	serialized, err := SerializeResponse(original)
	require.NoError(t, err)

	deserialized, err := DeserializeResponse(serialized)
	require.NoError(t, err)

	assert.Equal(t, original.Status, deserialized.Status)
	assert.Equal(t, original.TTLHours, deserialized.TTLHours)
	assert.Equal(t, original.Domain, deserialized.Domain)
	assert.Equal(t, original.DomainLength, deserialized.DomainLength)
}

func TestValidateHeaderInvalidVersion(t *testing.T) {
	header := &Header{
		Version:  0x11,
		Type:     TypeRequest,
		Length:   100,
		Reserved: 0,
	}

	_, err := SerializeHeader(header)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version")
}

func TestValidateRequestInvalidProtocol(t *testing.T) {
	req := &Request{
		Proto:      0xFF,
		NameLength: 4,
		Name:       "test",
	}

	_, err := SerializeRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid protocol")
}

func TestValidateRequestEmptyName(t *testing.T) {
	req := &Request{
		Proto:      ProtoHTTP,
		NameLength: 0,
		Name:       "",
	}

	_, err := SerializeRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "string field cannot be empty")
}

func TestIsValidType(t *testing.T) {
	assert.True(t, IsValidType(TypeRequest))
	assert.True(t, IsValidType(TypeResponse))
	assert.True(t, IsValidType(TypeError))
	assert.False(t, IsValidType(0xEE))
}
