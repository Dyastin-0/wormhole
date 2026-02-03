package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// HTTPLogSize is the fixed size of an HTTPLog's non-string fields in bytes (12 bytes).
const HTTPLogSize uint8 = 12

// HTTPLog represents a single HTTP request log entry.
type HTTPLog struct {
	// Timestamp represents when the request occurred in Unix seconds.
	Timestamp int64
	// Duration represents the request duration in microseconds.
	Duration uint32
}

// SerializeHTTPLog serializes an HTTPLog to a byte slice.
func SerializeHTTPLog(log *HTTPLog) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, HTTPLogSize))

	if err := binary.Write(buf, binary.BigEndian, log.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.Duration); err != nil {
		return nil, fmt.Errorf("failed to write duration: %w", err)
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// DeserializeHTTPLog deserializes a byte slice into an HTTPLog.
func DeserializeHTTPLog(data []byte) (*HTTPLog, error) {
	if len(data) < int(HTTPLogSize) {
		return nil, fmt.Errorf("http log data too small: expected at least %d bytes, got %d", HTTPLogSize, len(data))
	}

	reader := bytes.NewReader(data)
	var log HTTPLog

	if err := binary.Read(reader, binary.BigEndian, &log.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &log.Duration); err != nil {
		return nil, fmt.Errorf("failed to read duration: %w", err)
	}

	return &log, nil
}

// NewHTTPLog creates a new HTTPLog with the specified fields.
func NewHTTPLog(timestamp int64, duration uint32) *HTTPLog {
	return &HTTPLog{
		Timestamp: timestamp,
		Duration:  duration,
	}
}
