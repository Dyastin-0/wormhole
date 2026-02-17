package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
)

// HTTPLogFixedSize is the size of the fixed binary fields in bytes.
const HTTPLogFixedSize = 16 // int64 + uint32 + int32

// HTTPLog represents a single HTTP request/response log entry.
type HTTPLog struct {
	// Timestamp is when the request was forwarded, in Unix seconds.
	Timestamp int64
	// Duration is the server-measured request→response time in microseconds.
	Duration uint32
	// Status is the HTTP response status code.
	Status int32
	// RespSize is the size of the reponse.
	RespSize int64

	// Method is the HTTP request method (GET, POST, etc.).
	Method string
	// Path is the HTTP request path.
	Path string

	// ReqHeaders are the HTTP request headers.
	ReqHeaders http.Header
	// ReqBody is the request body, capped at maxInspectSize.
	ReqBody []byte

	// RespHeaders are the HTTP response headers.
	RespHeaders http.Header
	// RespBody is the response body, capped at maxInspectSize.
	RespBody []byte
}

// NewHTTPLog creates a new HTTPLog with the specified timing fields.
// Use the struct literal for the remaining fields.
func NewHTTPLog(timestamp int64, duration uint32) *HTTPLog {
	return &HTTPLog{
		Timestamp: timestamp,
		Duration:  duration,
	}
}

// SerializeHTTPLog serializes an HTTPLog into a byte slice.
//
// Layout:
//
//	[8] Timestamp (int64, big-endian)
//	[4] Duration  (uint32, big-endian)
//	[4] Status    (int32, big-endian)
//	[2] len(Method)
//	[n] Method
//	[2] len(Path)
//	[n] Path
//	[4] len(ReqBody)
//	[n] ReqBody
//	[4] len(RespBody)
//	[n] RespBody
//	[2] len(ReqHeaders) — number of keys
//	for each key:
//	  [2] len(key)
//	  [n] key
//	  [2] number of values
//	  for each value:
//	    [2] len(value)
//	    [n] value
//	[2] len(RespHeaders) — same layout as ReqHeaders
func SerializeHTTPLog(log *HTTPLog) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Fixed fields.
	if err := binary.Write(buf, binary.BigEndian, log.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.Duration); err != nil {
		return nil, fmt.Errorf("failed to write duration: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.Status); err != nil {
		return nil, fmt.Errorf("failed to write status: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, log.RespSize); err != nil {
		return nil, fmt.Errorf("failed to write response size: %w", err)
	}

	// Method and Path — bounded by uint16 (max 65535 bytes each, well within HTTP limits).
	if err := writeString16(buf, log.Method); err != nil {
		return nil, fmt.Errorf("failed to write method: %w", err)
	}
	if err := writeString16(buf, log.Path); err != nil {
		return nil, fmt.Errorf("failed to write path: %w", err)
	}

	// Bodies — bounded by uint32 (up to 4GiB, capped upstream at maxInspectSize).
	if err := writeBytes32(buf, log.ReqBody); err != nil {
		return nil, fmt.Errorf("failed to write req body: %w", err)
	}
	if err := writeBytes32(buf, log.RespBody); err != nil {
		return nil, fmt.Errorf("failed to write resp body: %w", err)
	}

	// Headers.
	if err := writeHeaders(buf, log.ReqHeaders); err != nil {
		return nil, fmt.Errorf("failed to write req headers: %w", err)
	}
	if err := writeHeaders(buf, log.RespHeaders); err != nil {
		return nil, fmt.Errorf("failed to write resp headers: %w", err)
	}

	return buf.Bytes(), nil
}

// DeserializeHTTPLog deserializes a byte slice produced by SerializeHTTPLog.
func DeserializeHTTPLog(data []byte) (*HTTPLog, error) {
	r := bytes.NewReader(data)
	log := &HTTPLog{}

	// Fixed fields.
	if err := binary.Read(r, binary.BigEndian, &log.Timestamp); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &log.Duration); err != nil {
		return nil, fmt.Errorf("failed to read duration: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &log.Status); err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &log.RespSize); err != nil {
		return nil, fmt.Errorf("failed to read response size: %w", err)
	}

	var err error

	if log.Method, err = readString16(r); err != nil {
		return nil, fmt.Errorf("failed to read method: %w", err)
	}
	if log.Path, err = readString16(r); err != nil {
		return nil, fmt.Errorf("failed to read path: %w", err)
	}

	if log.ReqBody, err = readBytes32(r); err != nil {
		return nil, fmt.Errorf("failed to read req body: %w", err)
	}
	if log.RespBody, err = readBytes32(r); err != nil {
		return nil, fmt.Errorf("failed to read resp body: %w", err)
	}

	if log.ReqHeaders, err = readHeaders(r); err != nil {
		return nil, fmt.Errorf("failed to read req headers: %w", err)
	}
	if log.RespHeaders, err = readHeaders(r); err != nil {
		return nil, fmt.Errorf("failed to read resp headers: %w", err)
	}

	return log, nil
}

// ---------------------------------------------------------------------------
// helpers — length-prefixed primitives
// ---------------------------------------------------------------------------

// writeString16 writes a uint16 length prefix followed by the string bytes.
// Strings longer than 65535 bytes are silently truncated — no HTTP method or
// path should ever approach that limit.
func writeString16(buf *bytes.Buffer, s string) error {
	b := []byte(s)
	if len(b) > 0xFFFF {
		b = b[:0xFFFF]
	}
	if err := binary.Write(buf, binary.BigEndian, uint16(len(b))); err != nil {
		return err
	}
	_, err := buf.Write(b)
	return err
}

func readString16(r *bytes.Reader) (string, error) {
	var length uint16
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}
	b := make([]byte, length)
	if _, err := r.Read(b); err != nil {
		return "", err
	}
	return string(b), nil
}

// writeBytes32 writes a uint32 length prefix followed by the bytes.
func writeBytes32(buf *bytes.Buffer, b []byte) error {
	if err := binary.Write(buf, binary.BigEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err := buf.Write(b)
	return err
}

func readBytes32(r *bytes.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	b := make([]byte, length)
	if _, err := r.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// writeHeaders encodes an http.Header as:
//
//	uint16  number of keys
//	for each key:
//	  uint16  len(key)  + key bytes
//	  uint16  number of values
//	  for each value:
//	    uint16  len(value) + value bytes
func writeHeaders(buf *bytes.Buffer, h http.Header) error {
	if err := binary.Write(buf, binary.BigEndian, uint16(len(h))); err != nil {
		return err
	}
	for key, vals := range h {
		if err := writeString16(buf, key); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(len(vals))); err != nil {
			return err
		}
		for _, v := range vals {
			if err := writeString16(buf, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func readHeaders(r *bytes.Reader) (http.Header, error) {
	var keyCount uint16
	if err := binary.Read(r, binary.BigEndian, &keyCount); err != nil {
		return nil, err
	}
	if keyCount == 0 {
		return nil, nil
	}
	h := make(http.Header, keyCount)
	for i := uint16(0); i < keyCount; i++ {
		key, err := readString16(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read header key: %w", err)
		}
		var valCount uint16
		if err := binary.Read(r, binary.BigEndian, &valCount); err != nil {
			return nil, fmt.Errorf("failed to read header value count: %w", err)
		}
		vals := make([]string, valCount)
		for j := uint16(0); j < valCount; j++ {
			vals[j], err = readString16(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read header value: %w", err)
			}
		}
		h[key] = vals
	}
	return h, nil
}
