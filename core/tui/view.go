package tui

import (
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
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.Title.Render(m.name),
		" ",
		m.spinner.View(),
	)

	lines := []string{header, ""}
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

	allKeys := keyMapSlice{
		m.logList.Keys().Up,
		m.logList.Keys().Down,
		m.logList.Keys().Enter,
		m.keys.Back,
		m.keys.Quit,
	}

	lines = append(lines, "", m.help.View(allKeys))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
