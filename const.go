package wormhole

import "errors"

var (
	// Used when client/server failed to accept an incoming connection
	ErrFailedToAcceptConn = errors.New("failed to accept conn")

	ErrFailedToCreateYamuxServer = errors.New("failed to create yamux server")
	ErrFailedToCreateYamuxClient = errors.New("failed to create yamux client")

	// Used when a json decoder failed to decode a json
	ErrFailedToDecodeMessage = errors.New("failed to decode message")
	// Used when a json encoder failed to encode a json
	ErrFailedToEncodeMessage = errors.New("failed to encode message")

	// Used when an unauthenticated client sent a used tunnel name along with status = 2
	ErrTunnelNameAlreadyUsed = errors.New("name already used")

	// Used when a client/server failed to dial a tcp server
	ErrFailedToDialTCP = errors.New("failed to dial tcp")
	// Used when a client/server failed to establish a tcp listener
	ErrFailedToListenToTCP = errors.New("failed to listen to tcp")
	// Used when a client/server failed to create a yamux stream
	ErrFailedToOpenStream = errors.New("failed to open stream")

	ErrHandshakeFailed = errors.New("handshake failed")
	// Used when a client sent an invalid protocol, along with status 3
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
	// Used when a client sent a invalid api key, along with status 1
	ErrUnauthenticated = errors.New("unauthenticated")

	// Used when the server failed to write http request to the yamux session
	ErrFailedToWriteHTTPRequestToTunnel = errors.New("failed to write http request to tunnel")
	// Used when the client failed to write the http request to the local http server
	ErrFailedToWriteHTTPRequest = errors.New("failed to write http request")
	// Used when the client failed to read response from the local http server
	ErrFailedToReadHTTPResponse = errors.New("failed to read http tunnel response")
	// Used when the client failed to send the response from the local http server
	// back to the server (wormhole server)
	ErrFailedToWriteHTTPResponse = errors.New("failed to write http tunnel response")
	// Used when the server failed to read the http response from the yamux session
	ErrFailedToReadHTTPResponseFromTunnel = errors.New("failed to read http response from tunnel")

	ErrContextCancelled = errors.New("context canceled")
	ErrNilContext       = errors.New("nil context")
	ErrNilTLSConfig     = errors.New("nil tls config")
	ErrNilDNSManager    = errors.New("nil dns manager")
	ErrNilStore         = errors.New("nil store")
)

const (
	// Used when a handshake succeed
	StatusOK = 0
	// Used when client a sent an invalid api key at handshake
	StatusUnauthenticated = 1
	// Used when an unauthenticated client sent a used tunnel name at handshake
	StatusNameAlreadyUsed = 2
	// Used when a client sent an invalid proto at handshake
	StatusUnsupportedProto = 3

	// Used at handshake and proceeding messages
	MaxJSONSize = 2048
)

const (
	// Used on ttl tunnel time out
	MsgTunnelttlTimeout = "tunnel ttl time out"
)

const (
	ProtoHTTP = "http"
	ProtoTCP  = "tcp"
)
