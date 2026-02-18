// Package proto defines the Wormhole binary protocol.
package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Header represents a Wormhole protocol message header.
type Header struct {
	// Length specifies the payload length in bytes (must not exceed MaxPayloadSize).
	Length uint64
	// Version specifies the protocol version (must be Version).
	Version uint8
	// Type specifies the message type (e.g., TypeRequest, TypeResponse).
	Type uint8
	// Flags specifies the
	Flags uint8
	// Reserved is a reserved field that must be zero.
	Reserved uint8
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

// SerializeHeader serializes a Header to a byte slice using a pooled buffer.
//
// Wire layout:
//
// [8] length 	(uint64, big-endian)
// [2] Version 	(uint8, big-endian)
// [2] Type 		(uint8, big-endian)
// [2] Flags 		(uint8, big-endian)
// [2] Reserved (uint8, big-endian)
func SerializeHeader(header *Header) ([]byte, error) {
	if err := validateHeader(header); err != nil {
		return nil, fmt.Errorf("header validation failed: %w", err)
	}

	bufPtr := headerBufferPool.Get().(*[]byte)
	defer headerBufferPool.Put(bufPtr)

	*bufPtr = (*bufPtr)[:0]
	buf := bytes.NewBuffer(*bufPtr)

	if err := binary.Write(buf, binary.BigEndian, header.Length); err != nil {
		return nil, fmt.Errorf("failed to write length: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.Version); err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.Type); err != nil {
		return nil, fmt.Errorf("failed to write type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, header.Flags); err != nil {
		return nil, fmt.Errorf("failed to write flags: %w", err)
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
