package metricstui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m metricsModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	if m.viewMode == ViewModeDetail {
		return m.viewDetail()
	}
	return m.viewList()
}

func (m metricsModel) viewList() string {
	title := titleStyle.Render(fmt.Sprintf("%s %s", m.name, m.spinner.View()))
	lines := []string{
		"",
		title,
		"",
	}

	if m.hasMetrics {
		lines = append(lines, m.renderMetrics()...)
	}

	if m.hasHTTPLogging && m.logStore.Len() > 0 {
		if m.hasMetrics {
			lines = append(lines, "")
		}
		lines = append(lines, m.renderHTTPLogList()...)
	}

	lines = append(lines, "")
	lines = append(lines, m.renderFooter()...)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m metricsModel) renderMetrics() []string {
	return []string{
		formatLine("Ingress", formatBytes(m.metricsData.current.Ingress), fmt.Sprintf("%s/s", formatBytes(uint64(m.metricsData.ingressRate)))),
		formatLine("Egress", formatBytes(m.metricsData.current.Egress), fmt.Sprintf("%s/s", formatBytes(uint64(m.metricsData.egressRate)))),
		"",
		formatLine("Active connections", fmt.Sprintf("%d", m.metricsData.current.ActiveConnections), ""),
		formatLine("Total connections", fmt.Sprintf("%d", m.metricsData.current.ConnectionCount), ""),
		"",
		formatLine("Uptime", formatDuration(time.Duration(m.metricsData.current.Uptime)), ""),
		formatLine("RTT", fmt.Sprintf("%.2f ms", float64(m.metricsData.current.RTT)/1000.0), ""),
	}
}

func (m metricsModel) renderHTTPLogList() []string {
	lines := []string{
		logHeaderStyle.Render(fmt.Sprintf("Requests (%d)", m.logStore.Len())),
		"",
	}

	visibleLogs, startIdx, endIdx := m.logStore.GetVisible()

	for i, log := range visibleLogs {
		actualIdx := startIdx + i
		var logLine string

		if actualIdx == m.logStore.selectedIndex {
			logLine = formatHTTPLogSelected(log)
		} else {
			logLine = formatHTTPLog(log)
		}

		lines = append(lines, logLine)
	}

	if m.logStore.Len() > m.logStore.maxVisibleLogs {
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, m.logStore.Len())
		lines = append(lines, "", footerStyle.Render(scrollInfo))
	}

	return lines
}

func (m metricsModel) renderFooter() []string {
	if m.logStore.Len() > 0 {
		return []string{footerStyle.Render("j/k: Navigate • Enter: View details • q: Quit")}
	}
	return []string{footerStyle.Render("Press q to quit")}
}
