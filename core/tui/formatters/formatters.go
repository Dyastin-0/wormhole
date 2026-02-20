// Package formatters provides human-readable formats for data.
package formatters

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/Dyastin-0/wormhole/stream"
	"github.com/charmbracelet/lipgloss"
)

const maxBodySize = 1 * 1024 * 1024

func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func FormatMetricLine(label, value, rate string) string {
	l := styles.Label.Render(label)
	v := styles.Value.Render(value)
	if rate != "" {
		r := styles.Rate.Render(rate)
		return lipgloss.JoinHorizontal(lipgloss.Left, l, v, " ", r)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, l, v)
}

func FormatHTTPLog(log *stream.HTTPEvent) string {
	timestamp := log.Start.Format("15:04:05")
	timeStr := styles.LogTime.Render(timestamp)
	methodStr := styles.LogMethod.Render(log.Method)

	var sizeStr string
	if log.RespSize > 0 {
		sizeStr = FormatBytes(uint64(log.RespSize))
	} else {
		sizeStr = FormatBytes(0)
	}
	sizeStr = styles.LogSize.Render(sizeStr)

	path := log.Path
	if len(path) > 40 {
		path = path[:37] + "..."
	}
	pathStr := styles.LogPath.Render(path)

	statusStr := FormatStatusCode(log.Status, false)

	durationMs := float64(log.Duration) / 1000.0
	durationStr := styles.LogDuration.Render(fmt.Sprintf("%.1f ms", durationMs))

	return lipgloss.JoinHorizontal(lipgloss.Left,
		timeStr, " ",
		methodStr, " ",
		statusStr, " ",
		pathStr, " ",
		sizeStr, " ",
		durationStr,
	)
}

func FormatHTTPLogSelected(log *stream.HTTPEvent) string {
	timestamp := log.Start.Format("15:04:05")

	var sizeStr string
	if log.RespSize > 0 {
		sizeStr = FormatBytes(uint64(log.RespSize))
	} else {
		sizeStr = FormatBytes(0)
	}

	path := log.Path
	if len(path) > 40 {
		path = path[:37] + "..."
	}

	durationMs := float64(log.Duration) / 1000.0

	timeStr := styles.LogTime.Background(styles.SelectedBG).Width(styles.TimeWidth + 1).Render(timestamp)
	methodStr := styles.LogMethod.Background(styles.SelectedBG).Width(styles.MethodWidth + 1).Render(log.Method)
	statusStr := FormatStatusCode(log.Status, true)
	pathStr := styles.LogPath.Background(styles.SelectedBG).Width(styles.PathWidth + 1).Render(path)
	sizeStrStyled := styles.LogSize.Background(styles.SelectedBG).Width(styles.SizeWidth + 1).Render(sizeStr)
	durationStr := styles.LogDuration.Background(styles.SelectedBG).Render(fmt.Sprintf("%.1f ms", durationMs))

	return lipgloss.JoinHorizontal(lipgloss.Left,
		timeStr,
		methodStr,
		statusStr,
		pathStr,
		sizeStrStyled,
		durationStr,
	)
}

func FormatStatusCode(code int, selected bool) string {
	var style lipgloss.Style

	switch {
	case code >= 500:
		style = styles.Status5xx
	case code >= 400:
		style = styles.Status4xx
	case code >= 300:
		style = styles.Status3xx
	case code >= 200:
		style = styles.Status2xx
	default:
		style = styles.Status1xx
	}

	if selected {
		style = style.Background(styles.SelectedBG).Width(styles.StatusWidth + 1)
	}

	return style.Render(fmt.Sprintf("%d", code))
}

func SortAndRenderHeaders(headers http.Header) string {
	var keys []string
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder

	for _, k := range keys {
		paddingCount := max(styles.HeaderKeyWidth-len(k)-1, 0)
		padding := strings.Repeat(" ", paddingCount)

		renderedKey := styles.Label.Width(0).Render(k)
		renderedColon := styles.Value.Width(0).Bold(true).Render(":")

		keyBlock := renderedKey + padding + renderedColon
		valStr := styles.Value.Align(lipgloss.Left).Width(styles.HeaderValueWidth).Render(strings.Join(headers[k], ", "))

		row := lipgloss.JoinHorizontal(lipgloss.Top, keyBlock, " ", valStr)
		sb.WriteString(row + "\n")
	}
	return sb.String()
}

func ReadResponseBody(resp *stream.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return bodyBytes, nil
}

func ReadRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, maxBodySize))
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	return body, nil
}

func FormatPercent(label string, pct float64) string {
	return fmt.Sprintf("%s %s",
		styles.HelpKey.Render(label),
		styles.Value.Width(5).Align(lipgloss.Right).Render(fmt.Sprintf("%3.0f%%", pct*100)),
	)
}
