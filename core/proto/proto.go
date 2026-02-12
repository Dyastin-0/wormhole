// Package proto defines the Wormhole binary protocol.
package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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
	MaxPayloadSize uint64 = 1024 * 1024
	// MaxStringLength is the maximum length for string fields like names or domains (4 KB).
	MaxStringLength uint32 = 4096
	// HeaderSize is the fixed size of a protocol header in bytes (12 bytes).
	HeaderSize uint8 = 12
	// RequestSize is the fixed size of a request's non-string fields in bytes (34 bytes).
	RequestSize = 34
	// ResponseSize is the fixed size of a response's non-string fields in bytes (15 bytes).
	ResponseSize uint8 = 15
	// MetricsSize is the fixed size of a metrics' fields in bytes (40).
	MetricsSize uint8 = 40
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

// Header represents a Wormhole protocol message header.
type Header struct {
	// Version specifies the protocol version (must be Version).
	Version uint8
	// Type specifies the message type (e.g., TypeRequest, TypeResponse).
	Type uint8
	// Flags specifies the
	Flags uint8
	// Length specifies the payload length in bytes (must not exceed MaxPayloadSize).
	Length uint64
	// Reserved is a reserved field that must be zero.
	Reserved uint8
}

// HasFlag checks if flag is set.
func (h *Header) HasFlag(flag uint8) bool {
	return h.Flags&flag != 0
}

// SetFlag sets a specific flag.
func (h *Header) SetFlag(flag uint8) {
	h.Flags |= flag
}

// ClearFlag clears a specific flag.
func (h *Header) ClearFlag(flag uint8) {
	h.Flags &^= flag
}

// Response represents the server's response to a tunnel request.
type Response struct {
	// Status indicates the result of the request (e.g., StatusOK, StatusNameTaken).
	Status uint8
	// TTLHours specifies the tunnel's lifetime in hours.
	TTLHours uint64
	// Port specifies the allocated TCP port for TCP tunnels (443 if not a TCP tunnel).
	Port uint16
	// DomainLength is the length of the Domain field in bytes (must not exceed MaxStringLength).
	DomainLength uint32
	// Domain is the assigned domain for the tunnel (e.g., "example.domain.com").
	Domain string
}

// SerializeHeader serializes a Header to a byte slice using a pooled buffer.
func SerializeHeader(header *Header) ([]byte, error) {
	if err := validateHeader(header); err != nil {
		return nil, fmt.Errorf("header validation failed: %w", err)
	}

	bufPtr := headerBufferPool.Get().(*[]byte)
	defer headerBufferPool.Put(bufPtr)

	*bufPtr = (*bufPtr)[:0]
	buf := bytes.NewBuffer(*bufPtr)

	if err := binary.Write(buf, binary.BigEndian, header.Version); err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.Type); err != nil {
		return nil, fmt.Errorf("failed to write type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.Flags); err != nil {
		return nil, fmt.Errorf("failed to write flags: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.Length); err != nil {
		return nil, fmt.Errorf("failed to write length: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.Reserved); err != nil {
		return nil, fmt.Errorf("failed to write reserved: %w", err)
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// DeserializeHeader deserializes a byte slice into a Header.
func DeserializeHeader(data []byte) (*Header, error) {
	if len(data) < int(HeaderSize) {
		return nil, ErrInvalidHeaderSize
	}

	reader := bytes.NewReader(data[:HeaderSize])
	var header Header

	if err := binary.Read(reader, binary.BigEndian, &header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if err := validateHeader(&header); err != nil {
		return nil, fmt.Errorf("header validation failed: %w", err)
	}

	return &header, nil
}

// SerializeResponse serializes a Response to a byte slice using a pooled buffer.
func SerializeResponse(resp *Response) ([]byte, error) {
	if err := validateResponse(resp); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	bufPtr := responseBufferPool.Get().(*[]byte)
	defer responseBufferPool.Put(bufPtr)

	*bufPtr = (*bufPtr)[:0]
	totalSize := int(ResponseSize) + len(resp.Domain)
	if cap(*bufPtr) < totalSize {
		*bufPtr = make([]byte, 0, totalSize)
	}
	buf := bytes.NewBuffer(*bufPtr)

	if err := binary.Write(buf, binary.BigEndian, resp.Status); err != nil {
		return nil, fmt.Errorf("failed to write status: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, resp.TTLHours); err != nil {
		return nil, fmt.Errorf("failed to write ttl: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, resp.Port); err != nil {
		return nil, fmt.Errorf("failed to write tcp port: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, resp.DomainLength); err != nil {
		return nil, fmt.Errorf("failed to write domain length: %w", err)
	}
	if _, err := buf.WriteString(resp.Domain); err != nil {
		return nil, fmt.Errorf("failed to write domain: %w", err)
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// DeserializeResponse deserializes a byte slice into a Response.
func DeserializeResponse(data []byte) (*Response, error) {
	if len(data) < int(ResponseSize) {
		return nil, ErrInvalidResponseSize
	}

	reader := bytes.NewReader(data)
	var resp Response

	if err := binary.Read(reader, binary.BigEndian, &resp.Status); err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &resp.TTLHours); err != nil {
		return nil, fmt.Errorf("failed to read TTL: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &resp.Port); err != nil {
		return nil, fmt.Errorf("failed to read tcp port: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &resp.DomainLength); err != nil {
		return nil, fmt.Errorf("failed to read domain length: %w", err)
	}

	expectedSize := int(ResponseSize) + int(resp.DomainLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	domainBytes := make([]byte, resp.DomainLength)
	if n, err := reader.Read(domainBytes); err != nil || n != int(resp.DomainLength) {
		return nil, fmt.Errorf("failed to read domain: %w", err)
	}
	resp.Domain = string(domainBytes)

	if err := validateResponse(&resp); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	return &resp, nil
}

// validateHeader validates a Header's fields.
func validateHeader(header *Header) error {
	if header.Version != Version {
		return ErrInvalidVersion
	}

	if header.Length > MaxPayloadSize {
		return ErrPayloadTooLarge
	}

	if !IsValidType(header.Type) {
		return ErrInvalidType
	}

	if header.Reserved != 0 {
		return ErrReservedFieldUsed
	}

	return nil
}

// validateResponse validates a Response's fields.
func validateResponse(resp *Response) error {
	switch resp.Status {
	case StatusOK, StatusInvalidURL, StatusNameTaken, StatusUnsupportedProto:
		// OK
	default:
		return ErrInvalidStatus
	}

	if resp.Status == StatusOK {
		if resp.DomainLength == 0 || len(resp.Domain) == 0 {
			return ErrEmptyString
		}

		if resp.DomainLength != uint32(len(resp.Domain)) {
			return ErrInvalidLength
		}

		if resp.DomainLength > MaxStringLength {
			return ErrStringTooLong
		}
	}

	return nil
}

// IsValidType checks if a message type is valid.
func IsValidType(msgType uint8) bool {
	switch msgType {
	case TypeRequest, TypeResponse, TypeAccess, TypePing, TypePong, TypeMetrics, TypeHTTPLog, TypeEnd, TypeError:
		return true
	default:
		return false
	}
}

// NewHeader creates a new Header with the specified message type and payload length.
func NewHeader(msgType uint8, length uint64) *Header {
	return &Header{
		Version:  Version,
		Type:     msgType,
		Flags:    0,
		Length:   length,
		Reserved: 0,
	}
}

// NewResponse creates a new Response with the specified status, TTL, and domain.
func NewResponse(status uint8, ttlHours uint64, domain string) *Response {
	return &Response{
		Status:       status,
		TTLHours:     ttlHours,
		Port:         0,
		DomainLength: uint32(len(domain)),
		Domain:       domain,
	}
}

// CalculateTunnelRequestSize calculates the total size of a serialized Request, including its header.
func CalculateTunnelRequestSize(req *Request) uint64 {
	return uint64(HeaderSize) + uint64(RequestSize) + uint64(len(req.Name))
}

// CalculateTunnelResponseSize calculates the total size of a serialized Response, including its header.
func CalculateTunnelResponseSize(resp *Response) uint64 {
	return uint64(HeaderSize) + uint64(ResponseSize) + uint64(len(resp.Domain))
}

func ProtoString(proto uint8) string {
	if proto == ProtoHTTP {
		return "http"
	}
	if proto == ProtoTCP {
		return "tcp"
	}
	if proto == ProtoTLS {
		return "tls"
	}
	return ""
}
