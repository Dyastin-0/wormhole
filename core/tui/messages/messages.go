// Package messages provides update messages for the components.
package messages

import (
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/stream"
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
	*proto.HTTPDurationLog
}

type HTTPEventMsg struct {
	*stream.HTTPEvent
}

type SetLogMsg struct {
	Log *stream.HTTPEvent
}

type LogSelectedMsg struct {
	Log *stream.HTTPEvent
}

type ViewModeChangeMsg struct {
	Mode ViewMode
}

type ViewMode int

const (
	ViewModeList ViewMode = iota
	ViewModeDetail
)
