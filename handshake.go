package wormhole

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/golang-jwt/jwt/v5"
)

const (
	StatusOK               = 0
	StatusUnauthenticated  = 1
	StatusIDAlreadyUsed    = 2
	StatusUnsupportedProto = 3

	MaxJSONSize = 2048
)

func (w *Wormhole) handshake(stream net.Conn) (*message, *jwt.MapClaims, error) {
	defer stream.Close()

	dec := json.NewDecoder(io.LimitReader(stream, MaxJSONSize))
	enc := json.NewEncoder(stream)

	dec.DisallowUnknownFields()

	var msg message

	err := dec.Decode(&msg)
	if err != nil {
		return nil, nil, fmt.Errorf("%v: %w", ErrFailedToDecodeMessage, err)
	}

	// validate api key if there is
	payload := &jwt.MapClaims{}
	// will decide how i will issue api tokens
	// if msg.APIKey != "" {
	// 	payload, err = w.Issuer.ParseAPIToken(msg.APIKey)
	// 	if err != nil {
	// 		errMsg := &message{Status: StatusUnauthenticated, Err: ErrUnauthenticated.Error()}
	//
	// 		// encode error to stream and return err
	// 		err = enc.Encode(errMsg)
	// 		if err != nil {
	// 			return nil, nil, errors.Join(
	// 				fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrUnauthenticated),
	// 				fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err),
	// 			)
	// 		}
	//
	// 		return nil, nil, fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrUnauthenticated)
	// 	}
	// }

	if _, exists := w.tunnels.Load(msg.TunnelName); exists {
		errMsg := &message{Status: StatusIDAlreadyUsed, Err: ErrTunnelNameAlreadyUsed.Error()}

		err = enc.Encode(errMsg)
		if err != nil {
			return nil, nil, errors.Join(
				fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrTunnelNameAlreadyUsed),
				fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err),
			)
		}

		return nil, nil, fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrTunnelNameAlreadyUsed)
	}

	if msg.TunnelProto != ProtoHTTP && msg.TunnelProto != ProtoTCP {
		errMsg := &message{Status: StatusUnsupportedProto, Err: ErrUnsupportedProtocol.Error()}

		err = enc.Encode(errMsg)
		if err != nil {
			return nil, nil, errors.Join(
				fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrUnsupportedProtocol),
				fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err),
			)
		}

		return nil, nil, ErrUnsupportedProtocol
	}

	err = enc.Encode(&message{
		TunnelID: msg.TunnelID,
		Status:   StatusOK,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("%v: %w", ErrHandshakeFailed, err)
	}

	return &msg, payload, nil
}
