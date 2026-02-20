package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// DurationLog correlates a request ID with its server-measured duration.
type HTTPDurationLog struct {
	// ID is the X-Wormhole-Request-ID UUID (always 36 bytes).
	ID string
	// Duration is the server-measured request→response time in microseconds.
	Duration uint64
}

const (
	uuidLen         = 36
	DurationLogSize = 8 + uuidLen
)

func NewHTTPDurationLog(uuid string, duration uint64) *HTTPDurationLog {
	return &HTTPDurationLog{
		ID:       uuid,
		Duration: duration,
	}
}

// SerializeDurationLog serializes a DurationLog into a fixed-size byte slice.
//
// Wire layout:
//
//	[8]  Duration (uint64, big-endian)
//	[36] ID       (fixed UTF-8, no length prefix)
func SerializeHTTPDurationLog(log *HTTPDurationLog) ([]byte, error) {
	if len(log.ID) != uuidLen {
		return nil, fmt.Errorf("invalid UUID length: got %d, want %d", len(log.ID), uuidLen)
	}

	buf := new(bytes.Buffer)
	buf.Grow(DurationLogSize)

	if err := binary.Write(buf, binary.BigEndian, log.Duration); err != nil {
		return nil, fmt.Errorf("failed to write duration: %w", err)
	}

	buf.WriteString(log.ID)

	return buf.Bytes(), nil
}

// DeserializeDurationLog deserializes a byte slice produced by SerializeDurationLog.
func DeserializeHTTPDurationLog(data []byte) (*HTTPDurationLog, error) {
	if len(data) != DurationLogSize {
		return nil, fmt.Errorf("invalid DurationLog size: got %d, want %d", len(data), DurationLogSize)
	}

	log := &HTTPDurationLog{}
	r := bytes.NewReader(data)

	if err := binary.Read(r, binary.BigEndian, &log.Duration); err != nil {
		return nil, fmt.Errorf("failed to read duration: %w", err)
	}

	id := make([]byte, uuidLen)
	if _, err := r.Read(id); err != nil {
		return nil, fmt.Errorf("failed to read ID: %w", err)
	}
	log.ID = string(id)

	return log, nil
}
