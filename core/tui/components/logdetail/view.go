package logdetail

import (
	"github.com/Dyastin-0/wormhole/core/tui/formatters"
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.log == nil {
		return "No request selected"
	}

	var title string
	var headerLines string

	switch m.activeTab {
	case messages.TabResponseBody:
		title = "Response Details"
		headerLines = formatters.SortAndRenderHeaders(m.log.Response.Header)
	case messages.TabRequestBody:
		title = "Request Details"
		headerLines = formatters.SortAndRenderHeaders(m.log.Request.Header)
	}

	meta := m.renderMetadata(title)
	footer := m.renderFooter()

	headerColumn := lipgloss.JoinVertical(lipgloss.Left,
		meta,
		headerLines,
		footer,
	)

	bodyColumn := m.renderBodyColumn()

	return lipgloss.JoinHorizontal(lipgloss.Left, headerColumn, "  ", bodyColumn)
}
