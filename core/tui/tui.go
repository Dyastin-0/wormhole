package tui

import (
	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/stream"
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
			case *stream.HTTPEvent:
				p.Send(messages.HTTPEventMsg{HTTPEvent: v})
			case *proto.HTTPDurationLog:
				p.Send(messages.HTTPLogMsg{HTTPDurationLog: v})
			default:
				p.Send(v)
			}
		}
	}()

	return p, metricsch
}
