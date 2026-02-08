// Package tui provides the wormhole TUI.
package tui

import (
	"net/http"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

func Start(
	name string,
	hasMetrics,
	hasHTTPLogging bool,
) (*tea.Program, chan any, chan *proto.HTTPLog, chan *http.Request) {
	metricsch := make(chan any, 16)
	model, httpLogch, requestch := New(name, hasMetrics, hasHTTPLogging)

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

	return p, metricsch, httpLogch, requestch
}
