package wormhole

import "fmt"

var (
	ErrFailedToReadConn                = fmt.Errorf("failed to read conn")
	ErrFailedToAcceptConn              = fmt.Errorf("failed to accept conn")
	ErrInvalidMessageFormat            = fmt.Errorf("invalid message format")
	ErrInvalidAction                   = fmt.Errorf("invalid action")
	ErrFailedToCreateYamuxServer       = fmt.Errorf("failed to create yamux server")
	ErrFailedToCreateYamuxClient       = fmt.Errorf("failed to create yamux client")
	ErrFailedToDecodeMessage           = fmt.Errorf("failed to decode message")
	ErrFailedToEncodeMessage           = fmt.Errorf("failed to encode message")
	ErrIDAlreadyUsed                   = fmt.Errorf("id already used")
	ErrFailedToDialTCP                 = fmt.Errorf("failed to dial tcp")
	ErrFailedToOpenStream              = fmt.Errorf("failed to open stream")
	ErrHandshakeFailed                 = fmt.Errorf("handshake failed")
	ErrUnsupportedProtocol             = fmt.Errorf("unsupported protocol")
	ErrFailedToWriteHTTPTunnelRequest  = fmt.Errorf("failed to tunnel http")
	ErrFailedToWriteTCPTunnelRequest   = fmt.Errorf("failed to tunnel tcp")
	ErrFailedToReadHTTPTunnelResponse  = fmt.Errorf("failed to read http tunnel response")
	ErrFailedToReadTCPTunnelResponse   = fmt.Errorf("failed to read tcp tunnel response")
	ErrFailedToWriteHTTPTunnelResponse = fmt.Errorf("failed to write http tunnel response")
	ErrFailedToWriteTCPTunnelResponse  = fmt.Errorf("failed to write tcp tunnel response")
)

const (
	httpProto = "http"
	tcpProto  = "tcp"
)
