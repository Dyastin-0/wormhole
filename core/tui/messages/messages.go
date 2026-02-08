// Package messages provides update messages for the components.
package messages

import (
	"net/http"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/stream"
)

type HTTPLogMsg struct {
	*proto.HTTPLog
	Request      *http.Request
	Response     *stream.Response
	ResponseBody []byte
	RequestBody  []byte
}

type MetricsMsg struct {
	Ingress           uint64
	Egress            uint64
	Uptime            uint64
	ConnectionCount   uint64
	ActiveConnections uint32
	RTT               uint32
}

type SetLogMsg struct {
	Log *HTTPLogMsg
}

type LogSelectedMsg struct {
	Log *HTTPLogMsg
}

type HTTPLogReadyMsg struct {
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

type Tab int

const (
	TabRequestBody Tab = iota
	TabResponseBody
)
