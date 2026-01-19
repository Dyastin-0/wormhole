package server

import (
	"bufio"
	"net"
	"strings"
	"time"
)

const (
	ProtoHTTP  = "http"
	ProtoTCP   = "tcp"
	ProtoHTTPS = "https"
	ProtoTLS   = "tls"
)

type ConnWithReader struct {
	net.Conn
	r *bufio.Reader
}

func (c *ConnWithReader) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

// Write writes to the underlying connection (bypassing the reader)
func (c *ConnWithReader) Write(p []byte) (int, error) {
	return c.Conn.Write(p)
}

// GetReader returns the underlying buffered reader
func (c *ConnWithReader) GetReader() *bufio.Reader {
	return c.r
}

type Sniff struct {
	peekN int
}

// Conn determines the underlying protocol of a network connection.
func (s *Sniff) Conn(conn net.Conn) (string, *bufio.Reader) {
	br := bufio.NewReader(conn)

	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	initialPeekSize := s.peekN
	if initialPeekSize == 0 || initialPeekSize > 512 {
		initialPeekSize = 64
	}

	peekedBytes, err := br.Peek(initialPeekSize)

	if err == nil && len(peekedBytes) >= initialPeekSize {
		if s.TLS(peekedBytes) {
			return ProtoTLS, br
		}
		if s.HTTP(peekedBytes) {
			return ProtoHTTP, br
		}

		if br.Buffered() > initialPeekSize {
			maxPeek := min(br.Buffered(), 512)
			peekedBytes, _ = br.Peek(maxPeek)
			if s.HTTP(peekedBytes) {
				return ProtoHTTP, br
			}
		}
	} else if len(peekedBytes) > 0 {
		if s.TLS(peekedBytes) {
			return ProtoTLS, br
		}
		if s.HTTP(peekedBytes) {
			return ProtoHTTP, br
		}
	}

	return ProtoTCP, br
}

// TLS determines if peekedBytes is a tls record.
func (s *Sniff) TLS(peekedBytes []byte) bool {
	if len(peekedBytes) < 5 {
		return false
	}

	// 0x16 = record type 'handshake'
	if peekedBytes[0] != 0x16 {
		return false
	}
	// Valid record layer versions:
	// 0x03 0x00 = SSL 3.0
	// 0x03 0x01 = TLS 1.0+
	// 0x03 0x02 = TLS 1.1
	// 0x03 0x03 = TLS 1.2
	if peekedBytes[1] != 0x03 {
		return false
	}
	if peekedBytes[2] > 0x04 {
		return false
	}
	length := uint16(peekedBytes[3])<<8 | uint16(peekedBytes[4])
	if length == 0 || length > 16384 {
		return false
	}
	return true
}

// HTTP determines if peekedBytes contains an http request.
func (s *Sniff) HTTP(peekedBytes []byte) bool {
	// GET / HTTP/1.1 -14 bytes without \r\n
	if len(peekedBytes) < 14 {
		return false
	}

	dataStr := string(peekedBytes)
	dataUpper := strings.ToUpper(dataStr)

	httpMethods := []string{
		"GET ", "POST ", "PUT ", "DELETE ", "HEAD ",
		"OPTIONS ", "PATCH ", "TRACE ", "CONNECT ",
	}

	for _, method := range httpMethods {
		if strings.HasPrefix(dataUpper, method) {
			if strings.Contains(dataUpper, "HTTP/1.") || strings.Contains(dataUpper, "HTTP/2") {
				return true
			}
			return false
		}
	}

	if strings.HasPrefix(dataStr, "PRI * HTTP/2.0") {
		return true
	}

	return false
}
