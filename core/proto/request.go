package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Request represents a client request to establish a tunnel.
type Request struct {
	// TTL is the desired tunnel time-to-live, ignored when APIKey is not present.
	TTL uint64
	// NameLength is the length of the Name field in bytes (must not exceed MaxStringLength).
	NameLength uint32
	// URLLength is the length of the URL field in bytes (must not exceed MaxStringLength).
	URLLength uint32
	// APIKeyLength is the length of the APIKey field in bytes (must not exceed MaxStringLength).
	APIKeyLength uint32
	// UsernameLength is the length of the AuthUsername field in bytes.
	UsernameLength uint32
	// PasswordLength is the length of the AuthPassword field in bytes.
	PasswordLength uint32
	// TokenLength is the length of the AuthToken field in bytes.
	TokenLength uint32
	// Name is the desired subdomain name for the tunnel (e.g., "example" for "example.domain.com").
	Name string
	// URL is the client's CNAME pointed to the tunnel endpoint.
	URL string
	// APIKey is the server-issued JWT token.
	APIKey string
	// AuthUsername is the username for HTTP basic auth.
	AuthUsername string
	// AuthPassword is the password for HTTP basic auth.
	AuthPassword string
	// AuthToken is the bearer token for bearer token auth.
	AuthToken string
	// Proto specifies the tunnel protocol (e.g., ProtoHTTP, ProtoTCP).
	Proto uint8
	// AuthType is the authentication type for the tunnel.
	AuthType uint8
}

// NewRequest creates a new Request with the specified protocol and subdomain name.
func NewRequest(proto uint8, name string, url string, ttl uint64, apiKey string) *Request {
	return &Request{
		Proto:        proto,
		TTL:          ttl,
		NameLength:   uint32(len(name)),
		Name:         name,
		URLLength:    uint32(len(url)),
		URL:          url,
		APIKeyLength: uint32(len(apiKey)),
		APIKey:       apiKey,
	}
}

// SerializeRequest serializes a Request to a byte slice using a pooled buffer.
//
// Wire layout:
//
// [1] Proto          (uint8,  big-endian)
// [8] TTL            (uint64, big-endian)
// [4] NameLength     (uint32, big-endian)
// [n] Name           ([]byte, UTF-8)
// [4] URLLength      (uint32, big-endian)
// [n] URL            ([]byte, UTF-8)
// [4] APIKeyLength   (uint32, big-endian)
// [n] APIKey         ([]byte, UTF-8)
// [1] AuthType       (uint8,  big-endian)
// [4] UsernameLength (uint32, big-endian)
// [n] AuthUsername   ([]byte, UTF-8)
// [4] PasswordLength (uint32, big-endian)
// [n] AuthPassword   ([]byte, UTF-8)
// [4] TokenLength    (uint32, big-endian)
// [n] AuthToken      ([]byte, UTF-8)
func SerializeRequest(req *Request) ([]byte, error) {
	req.NameLength = uint32(len(req.Name))
	req.APIKeyLength = uint32(len(req.APIKey))
	req.UsernameLength = uint32(len(req.AuthUsername))
	req.PasswordLength = uint32(len(req.AuthPassword))
	req.TokenLength = uint32(len(req.AuthToken))
	req.URLLength = uint32(len(req.URL))

	if err := validateRequest(req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	bufPtr := requestBufferPool.Get().(*[]byte)
	defer requestBufferPool.Put(bufPtr)

	*bufPtr = (*bufPtr)[:0]
	totalSize := int(RequestSize) + len(req.Name) + len(req.APIKey) +
		len(req.AuthUsername) + len(req.AuthPassword) + len(req.AuthToken)
	if cap(*bufPtr) < totalSize {
		*bufPtr = make([]byte, 0, totalSize)
	}
	buf := bytes.NewBuffer(*bufPtr)

	if err := binary.Write(buf, binary.BigEndian, req.Proto); err != nil {
		return nil, fmt.Errorf("failed to write protocol: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.TTL); err != nil {
		return nil, fmt.Errorf("failed to write ttl: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.NameLength); err != nil {
		return nil, fmt.Errorf("failed to write name length: %w", err)
	}
	if _, err := buf.WriteString(req.Name); err != nil {
		return nil, fmt.Errorf("failed to write name: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.URLLength); err != nil {
		return nil, fmt.Errorf("falied to write url length: %w", err)
	}
	if _, err := buf.WriteString(req.URL); err != nil {
		return nil, fmt.Errorf("failed to write url: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.APIKeyLength); err != nil {
		return nil, fmt.Errorf("failed to write api key length: %w", err)
	}
	if _, err := buf.WriteString(req.APIKey); err != nil {
		return nil, fmt.Errorf("failed to write api key: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.AuthType); err != nil {
		return nil, fmt.Errorf("failed to write auth type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.UsernameLength); err != nil {
		return nil, fmt.Errorf("failed to write username length: %w", err)
	}
	if _, err := buf.WriteString(req.AuthUsername); err != nil {
		return nil, fmt.Errorf("failed to write username: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.PasswordLength); err != nil {
		return nil, fmt.Errorf("failed to write password length: %w", err)
	}
	if _, err := buf.WriteString(req.AuthPassword); err != nil {
		return nil, fmt.Errorf("failed to write password: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, req.TokenLength); err != nil {
		return nil, fmt.Errorf("failed to write token length: %w", err)
	}
	if _, err := buf.WriteString(req.AuthToken); err != nil {
		return nil, fmt.Errorf("failed to write token: %w", err)
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
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
	if err := binary.Read(reader, binary.BigEndian, &req.TTL); err != nil {
		return nil, fmt.Errorf("failed to read ttl: %w", err)
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

	if err := binary.Read(reader, binary.BigEndian, &req.URLLength); err != nil {
		return nil, fmt.Errorf("failed to read url length: %w", err)
	}

	urlBytes := make([]byte, req.URLLength)
	if n, err := reader.Read(urlBytes); err != nil || n != int(req.URLLength) {
		return nil, fmt.Errorf("failed to read url: %w", err)
	}
	req.URL = string(urlBytes)

	if err := binary.Read(reader, binary.BigEndian, &req.APIKeyLength); err != nil {
		return nil, fmt.Errorf("failed to read api key length: %w", err)
	}

	expectedSize = int(RequestSize) + int(req.NameLength) + int(req.URLLength) + int(req.APIKeyLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	// Handle empty APIKey case
	if req.APIKeyLength > 0 {
		apiKeyBytes := make([]byte, req.APIKeyLength)
		if n, err := reader.Read(apiKeyBytes); err != nil || n != int(req.APIKeyLength) {
			return nil, fmt.Errorf("failed to read api key: %w", err)
		}
		req.APIKey = string(apiKeyBytes)
	} else {
		req.APIKey = ""
	}

	if err := binary.Read(reader, binary.BigEndian, &req.AuthType); err != nil {
		return nil, fmt.Errorf("failed to read auth type: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &req.UsernameLength); err != nil {
		return nil, fmt.Errorf("failed to read username length: %w", err)
	}

	expectedSize = int(RequestSize) + int(req.NameLength) + int(req.URLLength) + int(req.APIKeyLength) + int(req.UsernameLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	// Handle empty AuthUsername case
	if req.UsernameLength > 0 {
		usernameBytes := make([]byte, req.UsernameLength)
		if n, err := reader.Read(usernameBytes); err != nil || n != int(req.UsernameLength) {
			return nil, fmt.Errorf("failed to read username: %w", err)
		}
		req.AuthUsername = string(usernameBytes)
	} else {
		req.AuthUsername = ""
	}

	if err := binary.Read(reader, binary.BigEndian, &req.PasswordLength); err != nil {
		return nil, fmt.Errorf("failed to read password length: %w", err)
	}

	expectedSize = int(RequestSize) + int(req.NameLength) + int(req.URLLength) + int(req.APIKeyLength) + int(req.UsernameLength) + int(req.PasswordLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	// Handle empty AuthPassword case
	if req.PasswordLength > 0 {
		passwordBytes := make([]byte, req.PasswordLength)
		if n, err := reader.Read(passwordBytes); err != nil || n != int(req.PasswordLength) {
			return nil, fmt.Errorf("failed to read password: %w", err)
		}
		req.AuthPassword = string(passwordBytes)
	} else {
		req.AuthPassword = ""
	}

	if err := binary.Read(reader, binary.BigEndian, &req.TokenLength); err != nil {
		return nil, fmt.Errorf("failed to read token length: %w", err)
	}

	expectedSize = int(RequestSize) + int(req.NameLength) + int(req.URLLength) + int(req.APIKeyLength) + int(req.UsernameLength) + int(req.PasswordLength) + int(req.TokenLength)
	if len(data) < expectedSize {
		return nil, ErrInsufficientData
	}

	// Handle empty AuthToken case
	if req.TokenLength > 0 {
		tokenBytes := make([]byte, req.TokenLength)
		if n, err := reader.Read(tokenBytes); err != nil || n != int(req.TokenLength) {
			return nil, fmt.Errorf("failed to read token: %w", err)
		}
		req.AuthToken = string(tokenBytes)
	} else {
		req.AuthToken = ""
	}

	if err := validateRequest(&req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	return &req, nil
}

// validateRequest validates a Request's fields.
func validateRequest(req *Request) error {
	if req.Proto != ProtoHTTP && req.Proto != ProtoTCP && req.Proto != ProtoTLS {
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

	if req.APIKeyLength != uint32(len(req.APIKey)) {
		return ErrInvalidLength
	}

	if req.APIKeyLength > MaxStringLength {
		return ErrStringTooLong
	}

	if req.UsernameLength != uint32(len(req.AuthUsername)) {
		return ErrInvalidLength
	}

	if req.UsernameLength > MaxStringLength {
		return ErrStringTooLong
	}

	if req.PasswordLength != uint32(len(req.AuthPassword)) {
		return ErrInvalidLength
	}

	if req.PasswordLength > MaxStringLength {
		return ErrStringTooLong
	}

	if req.TokenLength != uint32(len(req.AuthToken)) {
		return ErrInvalidLength
	}

	if req.TokenLength > MaxStringLength {
		return ErrStringTooLong
	}

	return nil
}
