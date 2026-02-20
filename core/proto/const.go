package proto

import (
	"errors"
	"sync"
)

// Constants definition for protocol limits and sizes.
const (
	// MaxPayloadSize is the maximum allowed payload size for messages (1 MB).
	MaxPayloadSize = 1024 * 1024
	// MaxStringLength is the maximum length for string fields like names or domains (4 KB).
	MaxStringLength = 4096
)

// Constants definition of the protocol version.
const (
	// Version is the current protocol version (0x10).
	Version uint8 = 0x12
	// VERSION is the human-readable protocol version ("1.2").
	VERSION = "1.2"
)

// Errors returned by the Wormhole protocol.
var (
	// ErrInvalidVersion is returned when the protocol version is incorrect.
	ErrInvalidVersion = errors.New("invalid version")
	// ErrInvalidType is returned when an unknown message type is encountered.
	ErrInvalidType = errors.New("invalid type")
	// ErrReservedFieldUsed is returned when the reserved field is non-zero.
	ErrReservedFieldUsed = errors.New("reserved field must be zero")
	// ErrPayloadTooLarge is returned when the payload exceeds MaxPayloadSize.
	ErrPayloadTooLarge = errors.New("payload exceeds maximum size")
	// ErrStringTooLong is returned when a string field exceeds MaxStringLength.
	ErrStringTooLong = errors.New("string exceeds maximum length")
	// ErrInvalidHeaderSize is returned when the header data is too small.
	ErrInvalidHeaderSize = errors.New("header data too small")
	// ErrInvalidRequestSize is returned when the request data is too small.
	ErrInvalidRequestSize = errors.New("request data too small")
	// ErrInvalidResponseSize is returned when the response data is too small.
	ErrInvalidResponseSize = errors.New("response data too small")
	// ErrInvalidMetricsSize is returned when the metrics data is too small.
	ErrInvalidMetricsSize = errors.New("metrics data too small")
	// ErrInvalidLength is returned when a string length field does not match the actual string length.
	ErrInvalidLength = errors.New("invalid length field")
	// ErrInsufficientData is returned when there is not enough data to deserialize a string field.
	ErrInsufficientData = errors.New("insufficient data for string fields")
	// ErrEmptyString is returned when a required string field is empty.
	ErrEmptyString = errors.New("string field cannot be empty")
	// ErrInvalidProtocol is returned when an unsupported protocol is specified.
	ErrInvalidProtocol = errors.New("invalid protocol")
	// ErrInvalidStatus is returned when an unknown status code is encountered.
	ErrInvalidStatus = errors.New("invalid status code")
)

// Buffer pools for reusing byte slices across serialization calls.
var (
	headerBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 0, HeaderSize)
			return &buf
		},
	}

	requestBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 0, 512)
			return &buf
		},
	}

	responseBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 0, 512)
			return &buf
		},
	}

	metricsBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 0, MetricsSize)
			return &buf
		},
	}
)
