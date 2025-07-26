package wormhole

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/token"
	"github.com/golang-jwt/jwt/v5"
)

const (
	StatusOK               = 0
	StatusUnauthenticated  = 1
	StatusIDAlreadyUsed    = 2
	StatusUnsupportedProto = 3

	MaxJSONSize = 2048
)

func (w *Wormhole) handshake(enc *json.Encoder, dec *json.Decoder) (string, string, string, time.Duration, error) {
	var msg message

	err := dec.Decode(&msg)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("%v: %w", ErrFailedToDecodeMessage, err)
	}

	// validate api key if there is
	payload := &jwt.MapClaims{}
	// will decide how will i issue api tokens
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
			return "", "", "", 0, errors.Join(
				fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrTunnelNameAlreadyUsed),
				fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err),
			)
		}

		return "", "", "", 0, fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrTunnelNameAlreadyUsed)
	}

	if msg.TunnelProto != ProtoHTTP && msg.TunnelProto != ProtoTCP {
		errMsg := &message{Status: StatusUnsupportedProto, Err: ErrUnsupportedProtocol.Error()}

		err = enc.Encode(errMsg)
		if err != nil {
			return "", "", "", 0, errors.Join(
				fmt.Errorf("%v: %w", ErrHandshakeFailed, ErrUnsupportedProtocol),
				fmt.Errorf("%v: %w", ErrFailedToEncodeMessage, err),
			)
		}

		return "", "", "", 0, ErrUnsupportedProtocol
	}

	var ipv4, domain, proto string
	var ttl time.Duration

	if payload != nil && msg.TunnelID != "" {
		userID := (*payload)[token.PayloadID].(string)

		param := &db.GetTunnelParams{
			ID:     msg.TunnelID,
			UserID: userID,
		}

		res, err := w.Store.Tunnel.Get(w.ctx, param)
		if err != nil {
			w.Logger.Error(err.Error())
			return "", "", "", 0, err
		}

		domain = res.Domain
		ipv4 = res.Ipv4
		ttl = 24 * time.Hour
		proto = res.Protocol
	} else {
		domain = fmt.Sprintf("%s.%s", msg.TunnelName, w.DNSManager.API.BaseDNS())
		ipv4 = w.DNSManager.API.IPV4()
		ttl = 1 * time.Hour
		proto = msg.TunnelProto
	}

	err = enc.Encode(&message{
		Status:       StatusOK,
		TunnelDomain: domain,
	})
	if err != nil {
		return "", "", "", 0, fmt.Errorf("%v: %w", ErrHandshakeFailed, err)
	}

	return domain, proto, ipv4, ttl, nil
}
