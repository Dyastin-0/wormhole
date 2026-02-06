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
	m.keys.Search.SetEnabled(!m.searchMode)
	helpView := m.help.View(m.keys)

	var statusLine string
	if m.searchMode {
		matchInfo := ""
		if len(m.searchMatches) > 0 {
			matchInfo = fmt.Sprintf(" (%d/%d)", m.currentMatch+1, len(m.searchMatches))
		}
		searchStr := helpKeyStyle.Render("Search")
		statusLine = valueStyle.Width(0).Render(fmt.Sprintf(" /%s%s", m.searchQuery, matchInfo))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, searchStr, statusLine)
	} else if len(m.searchMatches) > 0 {
		matchStr := helpKeyStyle.Render("Match")
		statusLine = valueStyle.Width(0).Render(fmt.Sprintf(" %d/%d", m.currentMatch+1, len(m.searchMatches)))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, matchStr, statusLine)
	}

	if statusLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			"",
			statusLine,
			helpView,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, "", helpView)
}

func (m metricsModel) renderBodyColumn() string {
	viewports := lipgloss.JoinHorizontal(lipgloss.Left,
		m.viewport.View(),
		"  ",
		m.hexViewport.View(),
	)

	vertLabel := helpKeyStyle.Width(0).Render("Scroll-Y")
	vertVal := valueStyle.Width(0).Render(fmt.Sprintf("%3.0f%%", m.hexViewport.ScrollPercent()*100))

	horizLabel := helpKeyStyle.Width(0).Render("Scroll-X")
	horizVal := valueStyle.Width(0).Render(fmt.Sprintf("%3.0f%%", m.viewport.HorizontalScrollPercent()*100))
	separator := m.help.Styles.ShortSeparator.Render(" • ")
	scrollInfo := fmt.Sprintf("%s %s%s%s %s",
		vertLabel, vertVal,
		separator,
		horizLabel, horizVal,
	)

	body := lipgloss.JoinVertical(lipgloss.Left,
		valueStyle.Width(0).Render(fmt.Sprintf("Body %s", formatBytes(uint64(m.getBodySize())))),
		"",
		viewports,
		"",
		scrollInfo,
	)

	return body
}

func (m metricsModel) getBodySize() int {
	log := m.logStore.GetSelected()
	if log == nil {
		return 0
	}
	if m.activeTab == TabResponseBody {
		return len(log.responseBody)
	}
	return len(log.requestBody)
}

func (m metricsModel) formatDetailLineAligned(label, styledValue string, labelWidth int) string {
	l := detailLabelStyle.Render(label)
	leftAlignedLabel := lipgloss.PlaceHorizontal(labelWidth-1, lipgloss.Left, l)
	fullLabel := fmt.Sprintf("%s:", leftAlignedLabel)
	return lipgloss.JoinHorizontal(lipgloss.Left, fullLabel, " ", styledValue)
}
