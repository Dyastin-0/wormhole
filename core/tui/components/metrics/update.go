package metrics

import (
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.MetricsMsg:
		m.updateMetrics(msg)
	}
	return m, nil
}
