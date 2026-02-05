package metricstui

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Dyastin-0/wormhole/stream"
	"github.com/charmbracelet/lipgloss"
)

// formatBytes converts bytes to human-readable format.
func formatBytes(bytes uint64) string {
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

// formatDuration formats duration in a readable way.
func formatDuration(d time.Duration) string {
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

func formatLine(label, value, rate string) string {
	l := labelStyle.Render(label)
	v := valueStyle.Render(value)
	if rate != "" {
		r := rateStyle.Render(rate)
		return lipgloss.JoinHorizontal(lipgloss.Left, l, v, " ", r)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, l, v)
}

// formatHTTPLog formats a single HTTP log entry.
func formatHTTPLog(log *HTTPLogMsg) string {
	timestamp := time.Unix(log.Timestamp, 0).Format("15:04:05")
	timeStr := logTimeStyle.Render(timestamp)
	methodStr := logMethodStyle.Render(log.request.Method)

	var sizeStr string
	if log.response.Size > 0 {
		sizeStr = formatBytes(uint64(log.response.Size))
	} else {
		sizeStr = formatBytes(0)
	}
	sizeStr = logSizeStyle.Render(sizeStr)

	path := log.request.URL.Path
	if len(path) > 40 {
		path = path[:37] + "..."
	}
	pathStr := logPathStyle.Render(path)

	statusStr := formatStatusCode(log.response.StatusCode, false)

	durationMs := float64(log.Duration) / 1000.0
	durationStr := logDurationStyle.Render(fmt.Sprintf("%.1f ms", durationMs))

	return lipgloss.JoinHorizontal(lipgloss.Left,
		timeStr, " ",
		methodStr, " ",
		statusStr, " ",
		pathStr, " ",
		sizeStr, " ",
		durationStr,
	)
}

// formatHTTPLogSelected formats a selected HTTP log entry.
func formatHTTPLogSelected(log *HTTPLogMsg) string {
	timestamp := time.Unix(log.Timestamp, 0).Format("15:04:05")

	var sizeStr string
	if log.response.Size > 0 {
		sizeStr = formatBytes(uint64(log.response.Size))
	} else {
		sizeStr = formatBytes(0)
	}

	path := log.request.URL.Path
	if len(path) > 40 {
		path = path[:37] + "..."
	}

	durationMs := float64(log.Duration) / 1000.0

	timeStr := logTimeStyle.Background(selectedBG).Width(timeWidth + 1).Render(timestamp)
	methodStr := logMethodStyle.Background(selectedBG).Width(methodWidth + 1).Render(log.request.Method)
	statusStr := formatStatusCode(log.response.StatusCode, true)
	pathStr := logPathStyle.Background(selectedBG).Width(pathWidth + 1).Render(path)
	sizeStrStyled := logSizeStyle.Background(selectedBG).Width(sizeWidth + 1).Render(sizeStr)
	durationStr := logDurationStyle.Background(selectedBG).Render(fmt.Sprintf("%.1f ms", durationMs))

	return lipgloss.JoinHorizontal(lipgloss.Left,
		timeStr,
		methodStr,
		statusStr,
		pathStr,
		sizeStrStyled,
		durationStr,
	)
}

// formatStatusCode formats an HTTP status code with appropriate styling.
func formatStatusCode(code int, selected bool) string {
	var style lipgloss.Style
	if code >= 200 && code < 400 {
		style = logStatusOKStyle
	} else {
		style = logStatusErrorStyle
	}

	if selected {
		style = style.Background(selectedBG).Width(statusWidth + 1)
	}

	return style.Render(fmt.Sprintf("%d", code))
}

// sortAndRenderHeaders sorts and renders HTTP headers.
func sortAndRenderHeaders(headers http.Header) string {
	var keys []string
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder

	keyNameStyle := labelStyle.Width(0)
	colonStyle := lipgloss.NewStyle().Foreground(primary).Bold(true)
	valStyle := lipgloss.NewStyle().
		Width(headerValueWidth).
		Align(lipgloss.Left)

	for _, k := range keys {
		paddingCount := max(headerKeyWidth-len(k)-1, 0)
		padding := strings.Repeat(" ", paddingCount)

		renderedKey := keyNameStyle.Render(k)
		renderedColon := colonStyle.Render(":")

		keyBlock := renderedKey + padding + renderedColon
		valStr := valStyle.Render(strings.Join(headers[k], ", "))

		row := lipgloss.JoinHorizontal(lipgloss.Top, keyBlock, " ", valStr)
		sb.WriteString(row + "\n")
	}
	return sb.String()
}

// readResponseBody reads the response body up to maxBodySize.
func readResponseBody(resp *stream.Response) ([]byte, error) {
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

// readRequestBody reads the request body up to maxBodySize.
func readRequestBody(req *http.Request) ([]byte, error) {
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

func formatHexRows(data []byte, column int) string {
	if len(data) == 0 {
		return ""
	}

	hexChars := "0123456789abcdef"

	res := make([]byte, len(data)*3)

	writeIdx := 0
	for i, v := range data {
		res[writeIdx] = hexChars[v>>4]
		res[writeIdx+1] = hexChars[v&0x0f]

		if (i+1)%column == 0 {
			res[writeIdx+2] = '\n'
		} else {
			res[writeIdx+2] = ' '
		}
		writeIdx += 3
	}

	return string(res[:writeIdx])
}
