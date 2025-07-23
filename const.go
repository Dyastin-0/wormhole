package wormhole

import "errors"

var (
	ErrFailedToReadConn                = errors.New("failed to read conn")
	ErrFailedToAcceptConn              = errors.New("failed to accept conn")
	ErrInvalidMessageFormat            = errors.New("invalid message format")
	ErrInvalidAction                   = errors.New("invalid action")
	ErrFailedToCreateYamuxServer       = errors.New("failed to create yamux server")
	ErrFailedToCreateYamuxClient       = errors.New("failed to create yamux client")
	ErrFailedToDecodeMessage           = errors.New("failed to decode message")
	ErrFailedToEncodeMessage           = errors.New("failed to encode message")
	ErrTunnelNameAlreadyUsed           = errors.New("name already used")
	ErrFailedToDialTCP                 = errors.New("failed to dial tcp")
	ErrFailedToListenToTCP             = errors.New("failed to listen to tcp")
	ErrFailedToOpenStream              = errors.New("failed to open stream")
	ErrHandshakeFailed                 = errors.New("handshake failed")
	ErrUnsupportedProtocol             = errors.New("unsupported protocol")
	ErrUnauthenticated                 = errors.New("unauthenticated")
	ErrFailedToWriteHTTPTunnelRequest  = errors.New("failed to tunnel http")
	ErrFailedToWriteTCPTunnelRequest   = errors.New("failed to tunnel tcp")
	ErrFailedToReadHTTPTunnelResponse  = errors.New("failed to read http tunnel response")
	ErrFailedToReadTCPTunnelResponse   = errors.New("failed to read tcp tunnel response")
	ErrFailedToWriteHTTPTunnelResponse = errors.New("failed to write http tunnel response")
	ErrFailedToWriteTCPTunnelResponse  = errors.New("failed to write tcp tunnel response")
	ErrContextCancelled                = errors.New("context canceled")
	ErrNilContext                      = errors.New("nil context")
	ErrNilTLSConfig                    = errors.New("nil tls config")
	ErrNilDNSManager                   = errors.New("nil dns manager")
	ErrNilStore                        = errors.New("nil store")
)

const (
	ProtoHTTP = "http"
	ProtoTCP  = "tcp"
)
