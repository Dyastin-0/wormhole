package client

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	Timestamp int64
	Method    string
	Path      string
	Status    uint16
	Duration  uint32
}

type metricsModel struct {
	name        string
	spinner     spinner.Model
	metrics     MetricsMsg
	prevMetrics MetricsMsg
	lastUpdate  time.Time
	ingressRate float64
	egressRate  float64
	startTime   time.Time
	httpLogs    []HTTPLogMsg
}

var (
	subtle = lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	labelStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Width(20)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Align(lipgloss.Right).
			Width(15)

	rateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Align(lipgloss.Right).
			Width(15)

	logHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			MarginTop(1).
			MarginBottom(1)

	logTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Width(8)

	logMethodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Width(6)

	logPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	logStatusOKStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Width(5)

	logStatusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Width(5)

	logDurationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Align(lipgloss.Right).
				Width(10)
)

func newMetricsModel(name string) metricsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	return metricsModel{
		name:        name,
		spinner:     s,
		metrics:     MetricsMsg{},
		prevMetrics: MetricsMsg{},
		lastUpdate:  time.Now(),
		startTime:   time.Now(),
		httpLogs:    []HTTPLogMsg{},
	}
}

func (m metricsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
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

	case HTTPLogMsg:
		m.httpLogs = append(m.httpLogs, msg)
		if len(m.httpLogs) > 10 {
			m.httpLogs = m.httpLogs[len(m.httpLogs)-10:]
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m metricsModel) View() string {
	title := titleStyle.Render(fmt.Sprintf("%s %s", m.name, m.spinner.View()))

	lines := []string{
		title,
		"",
		m.formatLine("Ingress", formatBytes(m.metrics.Ingress), fmt.Sprintf("%s/s", formatBytes(uint64(m.ingressRate)))),
		m.formatLine("Egress", formatBytes(m.metrics.Egress), fmt.Sprintf("%s/s", formatBytes(uint64(m.egressRate)))),
		"",
		m.formatLine("Active connections", fmt.Sprintf("%d", m.metrics.ActiveConnections), ""),
		m.formatLine("Total connections", fmt.Sprintf("%d", m.metrics.ConnectionCount), ""),
		"",
		m.formatLine("Uptime", formatDuration(time.Duration(m.metrics.Uptime)), ""),
		m.formatLine("RTT", fmt.Sprintf("%.2f ms", float64(m.metrics.RTT)/1000.0), ""),
	}

	if len(m.httpLogs) > 0 {
		lines = append(lines, "")
		lines = append(lines, logHeaderStyle.Render("Requests"))

		for _, log := range m.httpLogs {
			lines = append(lines, m.formatHTTPLog(log))
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(subtle).Render("Press q to quit"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m metricsModel) formatLine(label, value, rate string) string {
	l := labelStyle.Render(label)
	v := valueStyle.Render(value)

	if rate != "" {
		r := rateStyle.Render(rate)
		return lipgloss.JoinHorizontal(lipgloss.Left, l, v, r)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, l, v)
}

func (m metricsModel) formatHTTPLog(log HTTPLogMsg) string {
	timestamp := time.Unix(log.Timestamp, 0).Format("15:04:05")
	timeStr := logTimeStyle.Render(timestamp)

	methodStr := logMethodStyle.Render(log.Method)

	path := log.Path
	if len(path) > 40 {
		path = path[:37] + "..."
	}
	pathStr := logPathStyle.Render(path)

	var statusStr string
	if log.Status >= 200 && log.Status < 300 {
		statusStr = logStatusOKStyle.Render(fmt.Sprintf("%d", log.Status))
	} else {
		statusStr = logStatusErrorStyle.Render(fmt.Sprintf("%d", log.Status))
	}

	durationMs := float64(log.Duration) / 1000.0
	durationStr := logDurationStyle.Render(fmt.Sprintf("%.1fms", durationMs))

	return fmt.Sprintf("%s  %s  %s  %s  %s", timeStr, methodStr, statusStr, pathStr, durationStr)
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
func StartMetricsDisplay(name string) (*tea.Program, chan<- any) {
	metricsChan := make(chan any, 10)

	p := tea.NewProgram(newMetricsModel(name))

	go func() {
		for msg := range metricsChan {
			p.Send(msg)
		}
	}()

	return p, metricsChan
}
