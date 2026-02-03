package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// HTTPLog represents a single HTTP request log entry.
type HTTPLog struct {
	// Timestamp represents when the request occurred in Unix seconds.
	Timestamp int64
	// MethodLength is the length of the Method field in bytes.
	MethodLength uint16
	// Method represents the HTTP method (GET, POST, etc.).
	Method string
	// PathLength is the length of the Path field in bytes.
	PathLength uint16
	// Path represents the request path.
	Path string
	// Status represents the HTTP response status code.
	Status uint16
	// Size represents the HTTP reponse content length.
	Size int64
	// Duration represents the request duration in microseconds.
	Duration uint32
}

// SerializeHTTPLog serializes an HTTPLog to a byte slice.
func SerializeHTTPLog(log *HTTPLog) ([]byte, error) {
	if err := validateHTTPLog(log); err != nil {
		return nil, fmt.Errorf("http log validation failed: %w", err)
	}

	totalSize := int(HTTPLogSize) + len(log.Method) + len(log.Path)
	buf := bytes.NewBuffer(make([]byte, 0, totalSize))

	if err := binary.Write(buf, binary.BigEndian, log.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.MethodLength); err != nil {
		return nil, fmt.Errorf("failed to write method length: %w", err)
	}
	if _, err := buf.WriteString(log.Method); err != nil {
		return nil, fmt.Errorf("failed to write method: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.PathLength); err != nil {
		return nil, fmt.Errorf("failed to write path length: %w", err)
	}
	if _, err := buf.WriteString(log.Path); err != nil {
		return nil, fmt.Errorf("failed to write path: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.Status); err != nil {
		return nil, fmt.Errorf("failed to write status: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.Size); err != nil {
		return nil, fmt.Errorf("failed to write size: %w", err)
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
	if err := binary.Read(reader, binary.BigEndian, &log.MethodLength); err != nil {
		return nil, fmt.Errorf("failed to read method length: %w", err)
	}

	expectedSize := int(HTTPLogSize) + int(log.MethodLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	methodBytes := make([]byte, log.MethodLength)
	if n, err := reader.Read(methodBytes); err != nil || n != int(log.MethodLength) {
		return nil, fmt.Errorf("failed to read method: %w", err)
	}
	log.Method = string(methodBytes)

	if err := binary.Read(reader, binary.BigEndian, &log.PathLength); err != nil {
		return nil, fmt.Errorf("failed to read path length: %w", err)
	}

	expectedSize = int(HTTPLogSize) + int(log.MethodLength) + int(log.PathLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	pathBytes := make([]byte, log.PathLength)
	if n, err := reader.Read(pathBytes); err != nil || n != int(log.PathLength) {
		return nil, fmt.Errorf("failed to read path: %w", err)
	}
	log.Path = string(pathBytes)

	if err := binary.Read(reader, binary.BigEndian, &log.Status); err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &log.Size); err != nil {
		return nil, fmt.Errorf("failed to read size: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &log.Duration); err != nil {
		return nil, fmt.Errorf("failed to read duration: %w", err)
	}

	if err := validateHTTPLog(&log); err != nil {
		return nil, fmt.Errorf("http log validation failed: %w", err)
	}

	return &log, nil
}

// validateHTTPLog validates an HTTPLog's fields.
func validateHTTPLog(log *HTTPLog) error {
	if log.MethodLength != uint16(len(log.Method)) {
		return ErrInvalidLength
	}

	if log.PathLength != uint16(len(log.Path)) {
		return ErrInvalidLength
	}

	if log.MethodLength > uint16(MaxStringLength) {
		return ErrStringTooLong
	}

	if log.PathLength > uint16(MaxStringLength) {
		return ErrStringTooLong
	}

	if len(log.Method) == 0 {
		return ErrEmptyString
	}

	if len(log.Path) == 0 {
		return ErrEmptyString
	}

	return nil
}

// NewHTTPLog creates a new HTTPLog with the specified fields.
func NewHTTPLog(timestamp int64, method, path string, status uint16, duration uint32, size int64) *HTTPLog {
	return &HTTPLog{
		Timestamp:    timestamp,
		MethodLength: uint16(len(method)),
		Method:       method,
		PathLength:   uint16(len(path)),
		Path:         path,
		Status:       status,
		Size:         size,
		Duration:     duration,
	}
}
