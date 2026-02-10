package stream

import (
	"io"
	"net"
	"strings"
	"time"
)

const (
	ProtoHTTP = "http"
	ProtoTCP  = "tcp"
	ProtoTLS  = "tls"
)

type Sniffer struct {
	peekN int
}

func Conn(conn net.Conn) (string, net.Conn) {
	sniffer := &Sniffer{peekN: 64}
	return sniffer.Conn(conn)
}

// Conn determines the underlying protocol of a network connection.
func (s *Sniffer) Conn(conn net.Conn) (string, net.Conn) {
	teeConn, teeReader := NewTeeConn(conn)

	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	initialPeekSize := s.peekN
	if initialPeekSize == 0 || initialPeekSize > 512 {
		initialPeekSize = 64
	}

	peekedBytes := make([]byte, initialPeekSize)
	n, err := io.ReadFull(teeReader, peekedBytes)

	if err == nil && n >= initialPeekSize {
		if s.TLS(peekedBytes) {
			return ProtoTLS, teeConn
		}
		if s.HTTP(peekedBytes) {
			return ProtoHTTP, teeConn
		}

		// Try reading more if needed
		morePeek := make([]byte, 512-initialPeekSize)
		n2, _ := teeReader.Read(morePeek)
		if n2 > 0 {
			combined := append(peekedBytes, morePeek[:n2]...)
			if s.HTTP(combined) {
				return ProtoHTTP, teeConn
			}
		}
	} else if n > 0 {
		peekedBytes = peekedBytes[:n]
		if s.TLS(peekedBytes) {
			return ProtoTLS, teeConn
		}
		if s.HTTP(peekedBytes) {
			return ProtoHTTP, teeConn
		}
	}

	return ProtoTCP, teeConn
}

// TLS determines if peekedBytes is a tls record.
func (s *Sniffer) TLS(peekedBytes []byte) bool {
	if len(peekedBytes) < 5 {
		return false
	}
	if peekedBytes[0] != 0x16 {
		return false
	}
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
func (s *Sniffer) HTTP(peekedBytes []byte) bool {
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
	return strings.HasPrefix(dataStr, "PRI * HTTP/2.0")
}
