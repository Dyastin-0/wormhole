package proto

import (
	"errors"
	"sync"
)

// Constants definition of message types for the Wormhole protocol.
const (
	// TypeRequest indicates a client request to establish a tunnel.
	TypeRequest uint8 = 0x01
	// TypeResponse indicates a server response to a tunnel request.
	TypeResponse uint8 = 0x02
	// TypeAccess indicates an incoming connection to an established tunnel.
	TypeAccess uint8 = 0x03
	// TypeMetrics indicates an incoming tunnel metrics stream.
	TypeMetrics = 0x05
	// TypeEnd indicates that a tunnel reached its end.
	TypeEnd uint8 = 0x06
	// TypePing indicates an incoming ping stream, all subsequent ping will be handled by it.
	TypePing uint8 = 0x07
	//  TypePong indicates an incoming pong message.
	TypePong uint8 = 0x08
	// TypeHTTPLog indicates an incoming http log stream, all subsequent logs will be handled by it.
	TypeHTTPLog uint8 = 0x09
	// TypeError indicates an error response from the server.
	TypeError uint8 = 0xFF
)

// Constants definition for HTTP authentication types.
const (
	// AuthTypeNone indicates no authentication.
	AuthTypeNone uint8 = 0x01
	// AuthTypeBasic implements a HTTP basic authentication.
	AuthTypeBasic uint8 = 0x02
	// AuthTypeBearer implements a bearer token authentication.
	AuthTypeBearer uint8 = 0x03
)

const (
	// FlagMetrics indicates that the client wants to stream the tunnel metrics.
	FlagMetrics = 0x01
	// FlagAllowHTTP indicates that the client explicitly allows HTTP requests regardless of protocol.
	FlagAllowHTTP = 0x02
	// FlagHTTPLog indicates that the client wants to receive HTTP request logs.
	FlagHTTPLog = 0x04
	// FlagTLSPassthrough indicates that the client wants to terminate TLS on its end.
	FlagTLSPassthrough = 0x08
)

// Constants definition for protocol limits and sizes.
const (
	// MaxPayloadSize is the maximum allowed payload size for messages (1 MB).
	MaxPayloadSize = 1024 * 1024
	// MaxStringLength is the maximum length for string fields like names or domains (4 KB).
	MaxStringLength = 4096
	// HeaderSize is the fixed size of a protocol header in bytes.
	HeaderSize = 12
	// RequestSize is the fixed size of a request's non-string fields in bytes .
	RequestSize = 34
	// ResponseSize is the fixed size of a response's non-string fields in bytes.
	ResponseSize = 15
	// MetricsSize is the fixed size of a metrics' fields in bytes.
	MetricsSize = 40
)

// Constants definition of supported protocols for tunneling.
const (
	// ProtoHTTP indicates an HTTP-based tunnel.
	ProtoHTTP uint8 = 0x01
	// ProtoTCP indicates a TCP-based tunnel.
	ProtoTCP uint8 = 0x02
	// ProtoTLS indicates a TLS wrapped TCP tunnel.
	ProtoTLS uint8 = 0x03
)

// Constants definition of response status codes.
const (
	// StatusOK indicates a successful tunnel creation.
	StatusOK uint8 = 0x01
	// StatusInvalidURL indicates that the URL paramater is invalid.
	StatusInvalidURL uint8 = 0x02
	// StatusNameTaken indicates the requested subdomain is already in use.
	StatusNameTaken uint8 = 0x03
	// StatusUnsupportedProto indicates the requested protocol is not supported.
	StatusUnsupportedProto uint8 = 0x04
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
