package tui

import (
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

func Start(
	name string,
	hasMetrics,
	hasHTTPLogging bool,
) (*tea.Program, chan any) {
	metricsch := make(chan any, 16)
	model := New(name, hasMetrics, hasHTTPLogging)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	go func() {
		for msg := range metricsch {
			switch v := msg.(type) {
			case messages.MetricsMsg:
				p.Send(v)
			default:
				p.Send(v)
			}
		}
	}()

	return p, metricsch
}
