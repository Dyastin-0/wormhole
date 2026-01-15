package server

import (
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

type Sniff struct {
	peekN int
}

// Conn determines the underlying protocol of a network connection.
func (s *Sniff) Conn(conn net.Conn) (string, *PeekableConn) {
	peekConn := NewPeekableConn(conn)
	peekConn.SetReadDeadline(time.Now().Add(5 * time.Second))

	peekedBytes, err := peekConn.Peek(s.peekN)
	if err != nil && len(peekedBytes) == 0 {
		return ProtoTCP, peekConn
	}

	peekConn.SetReadDeadline(time.Time{})

	if s.TLS(peekedBytes) {
		return ProtoTLS, peekConn
	}
	if s.HTTP(peekedBytes) {
		return ProtoHTTP, peekConn
	}
	return ProtoTCP, peekConn
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
	if len(peekedBytes) == 0 {
		return false
	}

	dataStr := strings.ToUpper(string(peekedBytes))
	httpMethods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH ", "TRACE ", "CONNECT "}

	for _, method := range httpMethods {
		if strings.HasPrefix(dataStr, method) {
			if strings.Contains(dataStr, "HTTP/1.") || strings.Contains(dataStr, "HTTP/2") {
				return true
			}
		}
	}

	return strings.HasPrefix(string(peekedBytes), "PRI * HTTP/2.0")
}
