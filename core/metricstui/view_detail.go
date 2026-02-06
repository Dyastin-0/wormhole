package metricstui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m metricsModel) viewDetail() string {
	log := m.logStore.GetSelected()
	if log == nil {
		return "No request selected"
	}

	var title string
	var headerLines string

	switch m.activeTab {
	case TabResponseBody:
		title = "Response Details"
		headerLines = sortAndRenderHeaders(log.response.Header)
	case TabRequestBody:
		title = "Request Details"
		headerLines = sortAndRenderHeaders(log.request.Header)
	}

	meta := m.renderDetailMetadata(log, title)
	footer := m.renderDetailFooter()

	headerColumn := lipgloss.JoinVertical(lipgloss.Left,
		meta,
		headerLines,
		footer,
	)

	bodyColumn := m.renderBodyColumn()

	return lipgloss.JoinHorizontal(lipgloss.Left, headerColumn, "  ", bodyColumn)
}

func (m metricsModel) renderDetailMetadata(log *HTTPLogMsg, title string) string {
	statusValue := formatStatusCode(log.response.StatusCode, false)

	metaLines := []string{
		title,
		"",
		m.formatDetailLineAligned(
			"Timestamp",
			logTimeStyle.Width(timeWidth+11).Render(
				time.Unix(log.Timestamp, 0).Format("2006-01-02 15:04:05"),
			),
			labelWidth,
		),
		m.formatDetailLineAligned(
			"Method",
			logMethodStyle.Render(log.request.Method),
			labelWidth,
		),
		m.formatDetailLineAligned(
			"Path",
			logPathStyle.Render(log.request.URL.Path),
			labelWidth,
		),
		m.formatDetailLineAligned(
			"Status",
			statusValue,
			labelWidth,
		),
		m.formatDetailLineAligned(
			"Size",
			logSizeStyle.Render(formatBytes(uint64(log.response.Size))),
			labelWidth,
		),
		m.formatDetailLineAligned(
			"Duration",
			logDurationStyle.Align(lipgloss.Left).Render(
				fmt.Sprintf("%.2f ms", float64(log.Duration)/1000.0),
			),
			labelWidth,
		),
		"",
	}

	return lipgloss.JoinVertical(lipgloss.Left, metaLines...)
}

func (m metricsModel) renderDetailFooter() string {
	var helpText string

	if m.searchMode {
		matchInfo := ""
		if len(m.searchMatches) > 0 {
			matchInfo = fmt.Sprintf(" (%d/%d)", m.currentMatch+1, len(m.searchMatches))
		}
		helpText = fmt.Sprintf("ESC: Clear/Back • Search: /%s%s", m.searchQuery, matchInfo)
	} else if len(m.searchMatches) > 0 {
		helpText = fmt.Sprintf("Match %d/%d • n/N: Next/Prev • /: Search • q: Quit",
			m.currentMatch+1, len(m.searchMatches))
	} else {
		helpText = "ESC: Back • j/k: Scroll • gg/G: Top/Bottom • /: Search • q: Quit"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		footerStyle.Render(helpText),
	)
}

func (m metricsModel) renderBodyColumn() string {
	viewports := lipgloss.JoinHorizontal(lipgloss.Left,
		m.viewport.View(),
		"  ",
		m.hexViewport.View(),
	)

	var size int
	switch m.activeTab {
	case TabResponseBody:
		size = len(m.logStore.GetSelected().responseBody)
	case TabRequestBody:
		size = len(m.logStore.GetSelected().requestBody)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("Body %s", formatBytes(uint64(size))),
		"",
		viewports,
		"",
		footerStyle.Render(
			fmt.Sprintf(
				"Scroll Y %3.f%% • Scroll X %3.f%%",
				m.hexViewport.ScrollPercent()*100,
				m.viewport.HorizontalScrollPercent()*100,
			),
		),
	)

	return body
}

func (m metricsModel) formatDetailLineAligned(label, styledValue string, labelWidth int) string {
	l := detailLabelStyle.Render(label)
	leftAlignedLabel := lipgloss.PlaceHorizontal(labelWidth-1, lipgloss.Left, l)
	fullLabel := fmt.Sprintf("%s:", leftAlignedLabel)
	return lipgloss.JoinHorizontal(lipgloss.Left, fullLabel, " ", styledValue)
}
