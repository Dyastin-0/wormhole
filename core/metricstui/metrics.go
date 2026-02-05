package metricstui

import (
	"net/http"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/stream"
)

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
	request      *http.Request
	response     *stream.Response
	responseBody []byte
	requestBody  []byte
}

// httpLogReadyMsg is sent when an HTTP log is ready to display.
type httpLogReadyMsg struct {
	log *HTTPLogMsg
}

type MetricsData struct {
	current     MetricsMsg
	previous    MetricsMsg
	lastUpdate  time.Time
	ingressRate float64
	egressRate  float64
}

func (md *MetricsData) UpdateMetrics(msg MetricsMsg) {
	now := time.Now()
	elapsed := now.Sub(md.lastUpdate).Seconds()

	if elapsed > 0 {
		ingressDiff := float64(msg.Ingress) - float64(md.previous.Ingress)
		egressDiff := float64(msg.Egress) - float64(md.previous.Egress)
		md.ingressRate = ingressDiff / elapsed
		md.egressRate = egressDiff / elapsed
	}

	md.previous = md.current
	md.current = msg
	md.lastUpdate = now
}

type HTTPLogStore struct {
	logs           []*HTTPLogMsg
	selectedIndex  int
	scrollOffset   int
	maxVisibleLogs int
	maxLogs        int
}

func NewHTTPLogStore(maxLogs, maxVisible int) *HTTPLogStore {
	return &HTTPLogStore{
		logs:           make([]*HTTPLogMsg, 0, maxLogs),
		selectedIndex:  0,
		scrollOffset:   0,
		maxVisibleLogs: maxVisible,
		maxLogs:        maxLogs,
	}
}

func (s *HTTPLogStore) AddLog(log *HTTPLogMsg) {
	s.logs = append(s.logs, log)

	if len(s.logs) > s.maxLogs {
		s.logs = s.logs[len(s.logs)-s.maxLogs:]
	}

	if s.selectedIndex == len(s.logs)-2 {
		s.selectedIndex = len(s.logs) - 1
	}

	if s.selectedIndex >= len(s.logs) {
		s.selectedIndex = len(s.logs) - 1
	}

	if s.selectedIndex >= s.scrollOffset+s.maxVisibleLogs {
		s.scrollOffset = s.selectedIndex - s.maxVisibleLogs + 1
	}
}

func (s *HTTPLogStore) MoveUp() {
	if len(s.logs) > 0 && s.selectedIndex > 0 {
		s.selectedIndex--
		if s.selectedIndex < s.scrollOffset {
			s.scrollOffset = s.selectedIndex
		}
	}
}

func (s *HTTPLogStore) MoveDown() {
	if len(s.logs) > 0 && s.selectedIndex < len(s.logs)-1 {
		s.selectedIndex++
		if s.selectedIndex >= s.scrollOffset+s.maxVisibleLogs {
			s.scrollOffset = s.selectedIndex - s.maxVisibleLogs + 1
		}
	}
}

func (s *HTTPLogStore) GetSelected() *HTTPLogMsg {
	if s.selectedIndex >= 0 && s.selectedIndex < len(s.logs) {
		return s.logs[s.selectedIndex]
	}
	return nil
}

func (s *HTTPLogStore) GetVisible() ([]*HTTPLogMsg, int, int) {
	if len(s.logs) == 0 {
		return nil, 0, 0
	}

	endIdx := min(s.scrollOffset+s.maxVisibleLogs, len(s.logs))
	return s.logs[s.scrollOffset:endIdx], s.scrollOffset, endIdx
}

func (s *HTTPLogStore) Len() int {
	return len(s.logs)
}
