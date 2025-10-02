// Package proto defines the Wormhole binary protocol.
package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Constants definition of message types for the Wormhole protocol.
const (
	// TypeRequest indicates a client request to establish a tunnel.
	TypeRequest uint8 = 0x01
	// TypeResponse indicates a server response to a tunnel request.
	TypeResponse uint8 = 0x02
	// TypeAccess indicates an incoming connection to an established tunnel.
	TypeAccess uint8 = 0x03
	// TypeAck indicates an acknowledgment of an access message.
	TypeAck uint8 = 0x04
	// TypeMetrics indicates an incoming tunnel metrics stream.
	TypeMetrics = 0x05
	// TypeEnd indicates that a tunnel reached its end.
	TypeEnd uint8 = 0x06
	// TypeError indicates an error response from the server.
	TypeError uint8 = 0xFF
)

// FlagMetrics indicates that the client wants to stream the tunnel metrics.
const FlagMetrics = 0x01

// Constants definition for protocol limits and sizes.
const (
	// MaxPayloadSize is the maximum allowed payload size for messages (1 MB).
	MaxPayloadSize uint64 = 1024 * 1024
	// MaxStringLength is the maximum length for string fields like names or domains (4 KB).
	MaxStringLength uint32 = 4096
	// HeaderSize is the fixed size of a protocol header in bytes (12 bytes).
	HeaderSize uint8 = 12
	// RequestSize is the fixed size of a request’s non-string fields in bytes (5 bytes).
	RequestSize uint8 = 5
	// ResponseSize is the fixed size of a response’s non-string fields in bytes (13 bytes).
	ResponseSize uint8 = 13
	// MetricsSize is the fixed size of a metrics' fields in bytes (36).
	MetricsSize uint8 = 36
)

// Constants definition of supported protocols for tunneling.
const (
	// ProtoHTTP indicates an HTTP-based tunnel.
	ProtoHTTP uint8 = 0x01
	// ProtoTCP indicates a TCP-based tunnel.
	ProtoTCP uint8 = 0x02
)

// Constants definition of response status codes.
const (
	// StatusOK indicates a successful tunnel creation.
	StatusOK uint8 = 0x01
	// StatusNameTaken indicates the requested subdomain is already in use.
	StatusNameTaken uint8 = 0x03
	// StatusUnsupportedProto indicates the requested protocol is not supported.
	StatusUnsupportedProto uint8 = 0x04
)

// Constants definition of the protocol version.
const (
	// Version is the current protocol version (0x10).
	Version uint8 = 0x10
	// VERSION is the human-readable protocol version ("1.0").
	VERSION = "1.0"
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

// Request represents a client’s request to establish a tunnel.
type Request struct {
	// Proto specifies the tunnel protocol (e.g., ProtoHTTP, ProtoTCP).
	Proto uint8
	// NameLength is the length of the Name field in bytes (must not exceed MaxStringLength).
	NameLength uint32
	// Name is the desired subdomain name for the tunnel (e.g., "example" for "example.domain.com").
	Name string
}

// Response represents the server’s response to a tunnel request.
type Response struct {
	// Status indicates the result of the request (e.g., StatusOK, StatusNameTaken).
	Status uint8
	// TTLHours specifies the tunnel’s lifetime in hours.
	TTLHours uint64
	// DomainLength is the length of the Domain field in bytes (must not exceed MaxStringLength).
	DomainLength uint32
	// Domain is the assigned domain for the tunnel (e.g., "example.domain.com").
	Domain string
}

// Metrics represets the tunnel's incoming and outgoing bytes metrics.
type Metrics struct {
	// Ingress represents the incoming bytes.
	Ingress uint64
	// Egress represents the outgoing bytes.
	Egress uint64
	// Uptime represents the time elapsed since tunnel started in milliseconds.
	Uptime uint64
	// ConnectionCount specifies the total number of connections.
	ConnectionCount uint64
	// ActiveConnections represents current active connections.
	ActiveConnections int32
}

// SerializeHeader serializes a Header to a byte slice.
func SerializeHeader(header *Header) ([]byte, error) {
	if err := validateHeader(header); err != nil {
		return nil, fmt.Errorf("header validation failed: %w", err)
	}

	buf := bytes.NewBuffer(make([]byte, 0, HeaderSize))

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

	return buf.Bytes(), nil
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

// SerializeRequest serializes a Request to a byte slice.
func SerializeRequest(req *Request) ([]byte, error) {
	if err := validateRequest(req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	totalSize := int(RequestSize) + len(req.Name)
	buf := bytes.NewBuffer(make([]byte, 0, totalSize))

	if err := binary.Write(buf, binary.BigEndian, req.Proto); err != nil {
		return nil, fmt.Errorf("failed to write protocol: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.NameLength); err != nil {
		return nil, fmt.Errorf("failed to write name length: %w", err)
	}
	if _, err := buf.WriteString(req.Name); err != nil {
		return nil, fmt.Errorf("failed to write name: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeRequest deserializes a byte slice into a Request.
func DeserializeRequest(data []byte) (*Request, error) {
	if len(data) < int(RequestSize) {
		return nil, ErrInvalidRequestSize
	}

	reader := bytes.NewReader(data)
	var req Request

	if err := binary.Read(reader, binary.BigEndian, &req.Proto); err != nil {
		return nil, fmt.Errorf("failed to read protocol: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &req.NameLength); err != nil {
		return nil, fmt.Errorf("failed to read name length: %w", err)
	}

	expectedSize := int(RequestSize) + int(req.NameLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	nameBytes := make([]byte, req.NameLength)
	if n, err := reader.Read(nameBytes); err != nil || n != int(req.NameLength) {
		return nil, fmt.Errorf("failed to read name: %w", err)
	}
	req.Name = string(nameBytes)

	if err := validateRequest(&req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	return &req, nil
}

// SerializeResponse serializes a Response to a byte slice.
func SerializeResponse(resp *Response) ([]byte, error) {
	if err := validateResponse(resp); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	totalSize := int(ResponseSize) + len(resp.Domain)
	buf := bytes.NewBuffer(make([]byte, 0, totalSize))

	if err := binary.Write(buf, binary.BigEndian, resp.Status); err != nil {
		return nil, fmt.Errorf("failed to write status: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, resp.TTLHours); err != nil {
		return nil, fmt.Errorf("failed to write ttl: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, resp.DomainLength); err != nil {
		return nil, fmt.Errorf("failed to write domain length: %w", err)
	}
	if _, err := buf.WriteString(resp.Domain); err != nil {
		return nil, fmt.Errorf("failed to write domain: %w", err)
	}

	return buf.Bytes(), nil
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

// SerializeMetrics serializes Metrics into byte slice.
func SerializeMetrics(metrics *Metrics) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, MetricsSize))

	if err := binary.Write(buf, binary.BigEndian, metrics.Ingress); err != nil {
		return nil, fmt.Errorf("failed to write ingress: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.Egress); err != nil {
		return nil, fmt.Errorf("failed to write egress: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.Uptime); err != nil {
		return nil, fmt.Errorf("failed to write uptime: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.ConnectionCount); err != nil {
		return nil, fmt.Errorf("failed to write connection count: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, metrics.ActiveConnections); err != nil {
		return nil, fmt.Errorf("failed to write active connections: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeMetrics deserializes a byte slice into Metrics.
func DeserializeMetrics(data []byte) (*Metrics, error) {
	if len(data) < int(MetricsSize) {
		return nil, ErrInvalidMetricsSize
	}

	reader := bytes.NewReader(data[:MetricsSize])
	var metrics Metrics

	if err := binary.Read(reader, binary.BigEndian, &metrics); err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	return &metrics, nil
}

// validateHeader validates a Header’s fields.
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

// validateRequest validates a Request’s fields.
func validateRequest(req *Request) error {
	if req.Proto != ProtoHTTP && req.Proto != ProtoTCP {
		return ErrInvalidProtocol
	}

	if req.NameLength == 0 || len(req.Name) == 0 {
		return ErrEmptyString
	}

	if req.NameLength != uint32(len(req.Name)) {
		return ErrInvalidLength
	}

	if req.NameLength > MaxStringLength {
		return ErrStringTooLong
	}

	return nil
}

// validateResponse validates a Response’s fields.
func validateResponse(resp *Response) error {
	switch resp.Status {
	case StatusOK, StatusNameTaken, StatusUnsupportedProto:
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
	case TypeRequest, TypeResponse, TypeAck, TypeAccess, TypeMetrics, TypeEnd, TypeError:
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

// NewRequest creates a new Request with the specified protocol and subdomain name.
func NewRequest(proto uint8, name string) *Request {
	return &Request{
		Proto:      proto,
		NameLength: uint32(len(name)),
		Name:       name,
	}
}

// NewResponse creates a new Response with the specified status, TTL, and domain.
func NewResponse(status uint8, ttlSeconds uint64, domain string) *Response {
	return &Response{
		Status:       status,
		TTLHours:     ttlSeconds,
		DomainLength: uint32(len(domain)),
		Domain:       domain,
	}
}

// NewMetrics creates a new Metrics with the specified ingress and egress.
func NewMetrics(ingress, egress, uptime, connectionCount uint64, activeConnections uint32) *Metrics {
	return &Metrics{
		Ingress:           ingress,
		Egress:            egress,
		Uptime:            uptime,
		ConnectionCount:   connectionCount,
		ActiveConnections: int32(activeConnections),
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
