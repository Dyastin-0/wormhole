package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestSerializeDeserialize(t *testing.T) {
	original := NewRequest(ProtoHTTP, "test-subdomain", "dyastin.dev", 0, "qweqweqe")

	serialized, err := SerializeRequest(original)
	require.NoError(t, err)

	deserialized, err := DeserializeRequest(serialized)
	require.NoError(t, err)

	require.Equal(t, original, deserialized)
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
