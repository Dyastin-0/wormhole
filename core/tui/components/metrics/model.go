// Package metrics implements the metrics component.
package metrics

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	enabled     bool
	data        Data
	lastUpdate  time.Time
	ingressRate float64
	egressRate  float64
}

type Data struct {
	Ingress           uint64
	Egress            uint64
	Uptime            uint64
	ConnectionCount   uint64
	ActiveConnections uint32
	RTT               uint32
}

func New(enabled bool) Model {
	return Model{
		enabled:    enabled,
		lastUpdate: time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) updateMetrics(msg messages.MetricsMsg) {
	now := time.Now()
	elapsed := now.Sub(m.lastUpdate).Seconds()

	if elapsed > 0 {
		ingressDiff := float64(msg.Ingress) - float64(m.data.Ingress)
		egressDiff := float64(msg.Egress) - float64(m.data.Egress)
		m.ingressRate = ingressDiff / elapsed
		m.egressRate = egressDiff / elapsed
	}

	m.data.Ingress = msg.Ingress
	m.data.Egress = msg.Egress
	m.data.Uptime = msg.Uptime
	m.data.ConnectionCount = msg.ConnectionCount
	m.data.ActiveConnections = msg.ActiveConnections
	m.data.RTT = msg.RTT

	m.lastUpdate = now
}
