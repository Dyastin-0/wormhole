package metrics

import (
	"fmt"
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/formatters"
)

func (m Model) View() string {
	if !m.enabled {
		return ""
	}

	return formatters.FormatMetricLine("Ingress", formatters.FormatBytes(m.data.Ingress), fmt.Sprintf("%s/s", formatters.FormatBytes(uint64(m.ingressRate)))) + "\n" +
		formatters.FormatMetricLine("Egress", formatters.FormatBytes(m.data.Egress), fmt.Sprintf("%s/s", formatters.FormatBytes(uint64(m.egressRate)))) + "\n" +
		"\n" +
		formatters.FormatMetricLine("Active connections", fmt.Sprintf("%d", m.data.ActiveConnections), "") + "\n" +
		formatters.FormatMetricLine("Total connections", fmt.Sprintf("%d", m.data.ConnectionCount), "") + "\n" +
		"\n" +
		formatters.FormatMetricLine("Uptime", formatters.FormatDuration(time.Duration(m.data.Uptime)), "") + "\n" +
		formatters.FormatMetricLine("RTT", fmt.Sprintf("%.2f ms", float64(m.data.RTT)/1000.0), "")
}
