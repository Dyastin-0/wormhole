// Package messages provides update messages for the components.
package messages

import (
	"github.com/Dyastin-0/wormhole/core/proto"
)

type MetricsMsg struct {
	Ingress           uint64
	Egress            uint64
	Uptime            uint64
	ConnectionCount   uint64
	ActiveConnections uint32
	RTT               uint32
}

type HTTPLogMsg struct {
	*proto.HTTPLog
}

type SetLogMsg struct {
	Log *HTTPLogMsg
}

type LogSelectedMsg struct {
	Log *HTTPLogMsg
}

type ViewModeChangeMsg struct {
	Mode ViewMode
}

type ViewMode int

const (
	ViewModeList ViewMode = iota
	ViewModeDetail
)
