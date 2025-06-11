package wormhole

import "fmt"

var (
	ErrFailedToReadConn          = fmt.Errorf("failed to read conn")
	ErrFailedToAcceptConn        = fmt.Errorf("failed to accept conn")
	ErrInvalidMessageFormat      = fmt.Errorf("invalid message format")
	ErrInvalidAction             = fmt.Errorf("invalid action")
	ErrFailedToCreateYamuxServer = fmt.Errorf("failed to create yamux server")
	ErrFailedToCreateYamuxClient = fmt.Errorf("failed to create yamux client")
	ErrFailedToDecodeMessage     = fmt.Errorf("failed to decode message")
	ErrFailedToEncodeMessage     = fmt.Errorf("failed to encode message")
	ErrIDAlreadyUsed             = fmt.Errorf("id already used")
	ErrFailedToDialWormhole      = fmt.Errorf("failed to dial wormhole")
	ErrFailedToOpenStream        = fmt.Errorf("failed to open stream")
	ErrHandshakeFailed           = fmt.Errorf("handshake failed")
)

const (
	StopAction = "stop"
	AddAction  = "add"
	PingAction = "ping"
)
