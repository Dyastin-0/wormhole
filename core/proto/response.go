package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Response represents the server's response to a tunnel request.
type Response struct {
	// TTLHours specifies the tunnel's lifetime in hours.
	TTLHours uint64
	// Domain is the assigned domain for the tunnel (e.g., "example.domain.com").
	Domain string
	// DomainLength is the length of the Domain field in bytes (must not exceed MaxStringLength).
	DomainLength uint32
	// Port specifies the allocated TCP port for TCP tunnels (443 if not a TCP tunnel).
	Port uint16
	// Status indicates the result of the request (e.g., StatusOK, StatusNameTaken).
	Status uint8
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

// SerializeResponse serializes a Response to a byte slice using a pooled buffer.
//
// Wire layout:
//
// [1] Status       (uint8,  big-endian)
// [8] TTLHours     (uint64, big-endian)
// [2] Port         (uint16, big-endian)
// [4] DomainLength (uint32, big-endian)
// [n] Domain       ([]byte, UTF-8)
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
