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
	ActiveConnections int32
}

type metricsModel struct {
	spinner     spinner.Model
	metrics     MetricsMsg
	prevMetrics MetricsMsg
	lastUpdate  time.Time
	ingressRate float64
	egressRate  float64
	startTime   time.Time
}

var (
	accent     = lipgloss.Color("250")
	highlight  = lipgloss.Color("255")
	subtle     = lipgloss.Color("245")
	borderTone = lipgloss.Color("240")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			Underline(true).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(subtle)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)

	rateStyle = lipgloss.NewStyle().
			Foreground(accent)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderTone).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)
)

func newMetricsModel() metricsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accent)

	return metricsModel{
		spinner:     s,
		metrics:     MetricsMsg{},
		prevMetrics: MetricsMsg{},
		lastUpdate:  time.Now(),
		startTime:   time.Now(),
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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m metricsModel) View() string {
	ingressLine := fmt.Sprintf("%s  %s  %s",
		labelStyle.Render("Ingress:"),
		valueStyle.Render(formatBytes(m.metrics.Ingress)),
		rateStyle.Render(fmt.Sprintf("(%s/s)", formatBytes(uint64(m.ingressRate)))),
	)

	egressLine := fmt.Sprintf("%s  %s  %s",
		labelStyle.Render("Egress: "),
		valueStyle.Render(formatBytes(m.metrics.Egress)),
		rateStyle.Render(fmt.Sprintf("(%s/s)", formatBytes(uint64(m.egressRate)))),
	)

	activeConnsLine := fmt.Sprintf("%s  %s",
		labelStyle.Render("Active: "),
		valueStyle.Render(fmt.Sprintf("%d", m.metrics.ActiveConnections)),
	)

	totalConnsLine := fmt.Sprintf("%s  %s",
		labelStyle.Render("Total:  "),
		valueStyle.Render(fmt.Sprintf("%d", m.metrics.ConnectionCount)),
	)

	uptimeLine := fmt.Sprintf("%s  %s",
		labelStyle.Render("Uptime: "),
		valueStyle.Render(formatDuration(time.Duration(m.metrics.Uptime)*time.Millisecond)),
	)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		labelStyle.Render("Traffic"),
		ingressLine,
		egressLine,
		"",
		labelStyle.Render("Connections"),
		activeConnsLine,
		totalConnsLine,
		"",
		uptimeLine,
		"",
		rateStyle.Render("Press q or ctrl+c to quit"),
	)

	contentWidth := lipgloss.Width(body)

	title := lipgloss.PlaceHorizontal(
		contentWidth,
		lipgloss.Center,
		titleStyle.Render("Tunnel Metrics"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left, title, body)

	return boxStyle.Render(content)
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
func StartMetricsDisplay() (*tea.Program, chan<- MetricsMsg) {
	metricsChan := make(chan MetricsMsg, 10)

	p := tea.NewProgram(newMetricsModel())

	go func() {
		for metrics := range metricsChan {
			p.Send(metrics)
		}
	}()

	return p, metricsChan
}
