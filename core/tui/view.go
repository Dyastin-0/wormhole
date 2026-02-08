package tui

import (
	"fmt"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	switch m.viewMode {
	case messages.ViewModeList:
		return m.viewList()
	case messages.ViewModeDetail:
		return m.logDetail.View()
	}
	return ""
}

func (m Model) viewList() string {
	title := styles.Title.Render(fmt.Sprintf("%s %s", m.name, m.spinner.View()))

	lines := []string{
		"",
		title,
		"",
	}

	metricsView := m.metrics.View()
	if metricsView != "" {
		lines = append(lines, metricsView)
	}

	logListView := m.logList.View()
	if logListView != "" {
		if metricsView != "" {
			lines = append(lines, "")
		}
		lines = append(lines, logListView)
	}

	lines = append(lines, "", m.help.View(m.keys))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
