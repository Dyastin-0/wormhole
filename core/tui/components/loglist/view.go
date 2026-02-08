package loglist

import (
	"fmt"

	"github.com/Dyastin-0/wormhole/core/tui/formatters"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.enabled || m.store.Len() == 0 {
		return ""
	}

	lines := []string{
		styles.Label.Render(fmt.Sprintf("Requests (%d)", m.store.Len())),
		"",
	}

	endIdx := min(m.scrollOffset+maxVisibleLogs, m.store.Len())
	visibleLogs := m.store.GetRange(m.scrollOffset, endIdx)

	for i, log := range visibleLogs {
		actualIdx := m.scrollOffset + i
		var logLine string

		if actualIdx == m.selectedIndex {
			logLine = formatters.FormatHTTPLogSelected(log)
		} else {
			logLine = formatters.FormatHTTPLog(log)
		}

		lines = append(lines, logLine)
	}

	if m.store.Len() > maxVisibleLogs {
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d", m.scrollOffset+1, endIdx, m.store.Len())
		lines = append(lines, "", styles.Footer.Render(scrollInfo))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
