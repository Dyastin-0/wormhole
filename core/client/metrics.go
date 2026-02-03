package client

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/stream"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewMode int

const (
	ViewModeList ViewMode = iota
	ViewModeDetail
)

const (
	maxLogs      = 1024
	headerHeight = 12
	footerHeight = 3
)

type MetricsMsg struct {
	Ingress           uint64
	Egress            uint64
	Uptime            uint64
	ConnectionCount   uint64
	ActiveConnections uint32
	RTT               uint32
}

type HTTPLogMsg struct {
	*proto.HTTPLog
	request   *http.Request
	response  *stream.Response
	bodyBytes []byte
}

type httpLogReadyMsg struct {
	log *HTTPLogMsg
}

type metricsModel struct {
	name           string
	spinner        spinner.Model
	viewport       viewport.Model
	metrics        MetricsMsg
	prevMetrics    MetricsMsg
	lastUpdate     time.Time
	ingressRate    float64
	egressRate     float64
	startTime      time.Time
	httpLogs       []*HTTPLogMsg
	hasMetrics     bool
	hasHTTPLogging bool
	ready          bool // Added to track if viewport is initialized

	viewMode       ViewMode
	selectedIndex  int
	scrollOffset   int
	maxVisibleLogs int

	httpLogch chan *proto.HTTPLog
	requestch chan *http.Request
}

const (
	timeWidth     = 8
	methodWidth   = 7
	statusWidth   = 3
	pathWidth     = 40
	sizeWidth     = 10
	durationWidth = 10
)

var (
	subtle  = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#808080"}
	primary = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}

	selectedBG = lipgloss.AdaptiveColor{Light: "#E6F3FF", Dark: "#1A3A52"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary)

	labelStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Width(20).
			Align(lipgloss.Left)

	valueStyle = lipgloss.NewStyle().
			Foreground(primary).
			Width(12).
			Align(lipgloss.Right)

	rateStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Width(12).
			Align(lipgloss.Right)

	logHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary)

	logTimeStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Width(timeWidth).
			Align(lipgloss.Left)

	logMethodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Width(methodWidth).
			Align(lipgloss.Left)

	logStatusOKStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Width(statusWidth)

	logStatusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Width(statusWidth)

	logPathStyle = lipgloss.NewStyle().
			Foreground(primary).
			Width(pathWidth).
			Align(lipgloss.Left)

	logSizeStyle = lipgloss.NewStyle().
			Foreground(primary).
			Width(sizeWidth).
			Align(lipgloss.Left)

	logDurationStyle = lipgloss.NewStyle().
				Foreground(subtle).
				Width(durationWidth).
				Align(lipgloss.Right)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(subtle).
				Bold(true)

	footerStyle = lipgloss.NewStyle().Foreground(subtle)
)

func newMetricsModel(
	name string, hasMetrics,
	hasHTTPLogging bool,
	httpLogch chan *proto.HTTPLog,
	requestch chan *http.Request,
) metricsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(subtle)

	return metricsModel{
		name:           name,
		spinner:        s,
		metrics:        MetricsMsg{},
		prevMetrics:    MetricsMsg{},
		lastUpdate:     time.Now(),
		startTime:      time.Now(),
		httpLogs:       make([]*HTTPLogMsg, 0, maxLogs),
		hasMetrics:     hasMetrics,
		hasHTTPLogging: hasHTTPLogging,
		viewMode:       ViewModeList,
		selectedIndex:  0,
		scrollOffset:   0,
		maxVisibleLogs: 10,
		httpLogch:      httpLogch,
		requestch:      requestch,
	}
}

func (m metricsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.viewMode == ViewModeDetail {
				m.viewMode = ViewModeList
				return m, nil
			}
		}

		switch m.viewMode {
		case ViewModeList:
			switch msg.String() {
			case "enter":
				if len(m.httpLogs) > 0 {
					m.viewMode = ViewModeDetail
					log := m.httpLogs[m.selectedIndex]
					content := string(log.bodyBytes)
					if len(content) == 0 {
						content = "No response body."
					}
					width := m.viewport.Width
					if width > 80 {
						content = lipgloss.NewStyle().Width(80).Render(content)
					}
					m.viewport.SetContent(content)
					m.viewport.GotoTop()
				}

			case "up", "k":
				if len(m.httpLogs) > 0 {
					if m.selectedIndex > 0 {
						m.selectedIndex--
						if m.selectedIndex < m.scrollOffset {
							m.scrollOffset = m.selectedIndex
						}
					}
				}

			case "down", "j":
				if len(m.httpLogs) > 0 {
					if m.selectedIndex < len(m.httpLogs)-1 {
						m.selectedIndex++
						if m.selectedIndex >= m.scrollOffset+m.maxVisibleLogs {
							m.scrollOffset = m.selectedIndex - m.maxVisibleLogs + 1
						}
					}
				}
			}
		case ViewModeDetail:
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case MetricsMsg:
		now := time.Now()
		elapsed := now.Sub(m.lastUpdate).Seconds()
		if elapsed > 0 {
			ingressDiff := float64(msg.Ingress) - float64(m.prevMetrics.Ingress)
			egressDiff := float64(msg.Egress) - float64(m.prevMetrics.Egress)
			m.ingressRate = ingressDiff / elapsed
			m.egressRate = egressDiff / elapsed
		}
		m.prevMetrics = m.metrics
		m.metrics = msg
		m.lastUpdate = now

	case *stream.Response:
		bodyBytes, err := readResponseBody(msg)
		if err != nil {
			bodyBytes = fmt.Appendf(nil, "Error reading body: %v", err)
		}

		return m, func() tea.Msg {
			var protoLog *proto.HTTPLog

			select {
			case protoLog = <-m.httpLogch:
			case <-time.After(30 * time.Second):
				return nil
			}

			return httpLogReadyMsg{
				log: &HTTPLogMsg{
					HTTPLog:   protoLog,
					response:  msg,
					bodyBytes: bodyBytes,
				},
			}
		}

	case httpLogReadyMsg:
		if msg.log == nil {
			return m, nil
		}

		m.httpLogs = append(m.httpLogs, msg.log)

		if len(m.httpLogs) > maxLogs {
			m.httpLogs = m.httpLogs[len(m.httpLogs)-maxLogs:]
		}

		if m.selectedIndex == len(m.httpLogs)-2 {
			m.selectedIndex = len(m.httpLogs) - 1
		}

		if m.selectedIndex >= len(m.httpLogs) {
			m.selectedIndex = len(m.httpLogs) - 1
		}

		if m.selectedIndex >= m.scrollOffset+m.maxVisibleLogs {
			m.scrollOffset = m.selectedIndex - m.maxVisibleLogs + 1
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

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
		lines = append(lines,
			m.formatLine("Ingress", formatBytes(m.metrics.Ingress), fmt.Sprintf("%s/s", formatBytes(uint64(m.ingressRate)))),
			m.formatLine("Egress", formatBytes(m.metrics.Egress), fmt.Sprintf("%s/s", formatBytes(uint64(m.egressRate)))),
			"",
			m.formatLine("Active connections", fmt.Sprintf("%d", m.metrics.ActiveConnections), ""),
			m.formatLine("Total connections", fmt.Sprintf("%d", m.metrics.ConnectionCount), ""),
			"",
			m.formatLine("Uptime", formatDuration(time.Duration(m.metrics.Uptime)), ""),
			m.formatLine("RTT", fmt.Sprintf("%.2f ms", float64(m.metrics.RTT)/1000.0), ""),
		)
	}

	if m.hasHTTPLogging && len(m.httpLogs) > 0 {
		if m.hasMetrics {
			lines = append(lines, "")
		}
		lines = append(lines, logHeaderStyle.Render(fmt.Sprintf("Requests (%d)", len(m.httpLogs))))
		lines = append(lines, "")

		endIdx := min(m.scrollOffset+m.maxVisibleLogs, len(m.httpLogs))

		for i := m.scrollOffset; i < endIdx; i++ {
			log := m.httpLogs[i]
			var logLine string

			if i == m.selectedIndex {
				logLine = m.formatHTTPLogSelected(log)
			} else {
				logLine = m.formatHTTPLog(log)
			}

			lines = append(lines, logLine)
		}

		if len(m.httpLogs) > m.maxVisibleLogs {
			scrollInfo := fmt.Sprintf("Showing %d-%d of %d", m.scrollOffset+1, endIdx, len(m.httpLogs))
			lines = append(lines, "", footerStyle.Render(scrollInfo))
		}
	}

	lines = append(lines, "")
	if len(m.httpLogs) > 0 {
		lines = append(lines, footerStyle.Render("j/k: Navigate • Enter: View details • q: Quit"))
	} else {
		lines = append(lines, footerStyle.Render("Press q to quit"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m metricsModel) viewDetail() string {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.httpLogs) {
		return "No request selected"
	}

	log := m.httpLogs[m.selectedIndex]

	title := titleStyle.Render("Request Details")

	labelWidth := 12

	var statusValue string
	if log.response.StatusCode >= 200 && log.response.StatusCode < 400 {
		statusValue = logStatusOKStyle.Render(fmt.Sprintf("%d", log.response.StatusCode))
	} else {
		statusValue = logStatusErrorStyle.Render(fmt.Sprintf("%d", log.response.StatusCode))
	}

	headerLines := []string{
		"",
		title,
		"",
		m.formatDetailLineAligned("Timestamp", logTimeStyle.Width(timeWidth+11).Render(time.Unix(log.Timestamp, 0).Format("2006-01-02 15:04:05")), labelWidth),
		m.formatDetailLineAligned("Method", logMethodStyle.Render(log.request.Method), labelWidth),
		m.formatDetailLineAligned("Path", logPathStyle.Render(log.request.URL.Path), labelWidth),
		m.formatDetailLineAligned("Status", statusValue, labelWidth),
		m.formatDetailLineAligned("Size", logSizeStyle.Render(formatBytes(uint64(log.response.Size))), labelWidth),
		m.formatDetailLineAligned("Duration", logDurationStyle.Align(lipgloss.Left).Render(fmt.Sprintf("%.2f ms", float64(log.Duration)/1000.0)), labelWidth),
		"",
	}

	header := lipgloss.JoinVertical(lipgloss.Left, headerLines...)

	footer := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100),
		"",
		footerStyle.Render("ESC: go back • j/k: scroll • q: Quit"),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.viewport.View(),
		footer,
	)
}

func (m metricsModel) formatDetailLineAligned(label, styledValue string, labelWidth int) string {
	l := detailLabelStyle.
		Width(labelWidth).
		Align(lipgloss.Right).
		Render(label + ":")

	return lipgloss.JoinHorizontal(lipgloss.Left, l, " ", styledValue)
}

func (m metricsModel) formatLine(label, value, rate string) string {
	l := labelStyle.Render(label)
	v := valueStyle.Render(value)
	if rate != "" {
		r := rateStyle.Render(rate)
		return lipgloss.JoinHorizontal(lipgloss.Left, l, v, " ", r)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, l, v)
}

func (m metricsModel) formatHTTPLogSelected(log *HTTPLogMsg) string {
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

	var statusStr string
	if log.response.StatusCode >= 200 && log.response.StatusCode < 400 {
		statusStr = logStatusOKStyle.Background(selectedBG).Width(statusWidth + 1).Render(fmt.Sprintf("%d", log.response.StatusCode))
	} else {
		statusStr = logStatusErrorStyle.Background(selectedBG).Width(statusWidth + 1).Render(fmt.Sprintf("%d", log.response.StatusCode))
	}

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

func (m metricsModel) formatHTTPLog(log *HTTPLogMsg) string {
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

	var statusStr string
	if log.response.StatusCode >= 200 && log.response.StatusCode < 400 {
		statusStr = logStatusOKStyle.Render(fmt.Sprintf("%d", log.response.StatusCode))
	} else {
		statusStr = logStatusErrorStyle.Render(fmt.Sprintf("%d", log.response.StatusCode))
	}

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

// StartMetricsDisplay starts the metrics display UI.
func StartMetricsDisplay(name string, hasMetrics, hasHTTPLogging bool) (*tea.Program, chan any, chan *proto.HTTPLog, chan *http.Request) {
	metricsch := make(chan any, 16)
	httpLogch := make(chan *proto.HTTPLog, 16)
	requestch := make(chan *http.Request, 16)
	p := tea.NewProgram(newMetricsModel(name, hasMetrics, hasHTTPLogging, httpLogch, requestch), tea.WithAltScreen())

	go func() {
		for msg := range metricsch {
			p.Send(msg)
		}
	}()

	return p, metricsch, httpLogch, requestch
}

func readResponseBody(resp *stream.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}

	const maxBodySize = 1024 * 1024
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return bodyBytes, nil
}
