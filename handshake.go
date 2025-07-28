package wormhole

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
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
	var err error
	err = dec.Decode(&msg)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("%v: %w", ErrFailedToDecodeMessage, err)
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

	domain = fmt.Sprintf("%s.%s", msg.TunnelName, w.DNSManager.API.BaseDNS())
	ipv4 = w.DNSManager.API.IPV4()
	ttl = 1 * time.Hour
	proto = msg.TunnelProto

	if _, exists := w.tunnels.Load(domain); exists {
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

	err = enc.Encode(&message{
		Status:       StatusOK,
		TunnelDomain: domain,
	})
	if err != nil {
		return "", "", "", 0, fmt.Errorf("%v: %w", ErrHandshakeFailed, err)
	}

	return domain, proto, ipv4, ttl, nil
}
