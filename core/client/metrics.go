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
	name        string
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
			MarginBottom(1).Width(42)
)

const (
	labelWidth = 10
	valueWidth = 10
	rateWidth  = 12
)

func newMetricsModel(name string) metricsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accent)

	return metricsModel{
		name:        name,
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
	ingressLine := formatLineRate("Ingress:", m.metrics.Ingress, m.ingressRate)
	egressLine := formatLineRate("Egress:", m.metrics.Egress, m.egressRate)

	activeConnsLine := formatLine("Active:", fmt.Sprintf("%d", m.metrics.ActiveConnections))
	totalConnsLine := formatLine("Total:", fmt.Sprintf("%d", m.metrics.ConnectionCount))

	uptimeLine := formatLine("Uptime:", formatDuration(time.Duration(m.metrics.Uptime)))

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
		labelStyle.Render("Press q or ctrl+c to quit"),
	)

	innerWidth := boxStyle.GetWidth() -
		boxStyle.GetPaddingLeft() - boxStyle.GetPaddingRight()

	title := lipgloss.PlaceHorizontal(
		innerWidth,
		lipgloss.Center,
		titleStyle.Render(m.name),
	)

	centeredBody := lipgloss.PlaceHorizontal(
		innerWidth,
		lipgloss.Center,
		body,
	)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		centeredBody,
	)

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

func formatLineRate(label string, value uint64, rate float64) string {
	labelText := fmt.Sprintf("%-*s", labelWidth, label)
	valueText := fmt.Sprintf("%-*s", valueWidth, formatBytes(value))
	rateText := fmt.Sprintf("%-*s", rateWidth, fmt.Sprintf("(%s/s)", formatBytes(uint64(rate))))

	return labelStyle.Render(labelText) +
		valueStyle.Render(valueText) +
		rateStyle.Render(rateText)
}

func formatLine(label string, value string) string {
	labelText := fmt.Sprintf("%-*s", labelWidth, label)
	valueText := fmt.Sprintf("%-*s", valueWidth, value)

	return labelStyle.Render(labelText) + valueStyle.Render(valueText)
}

// StartMetricsDisplay starts the metrics display UI.
func StartMetricsDisplay(name string) (*tea.Program, chan<- MetricsMsg) {
	metricsChan := make(chan MetricsMsg, 10)

	p := tea.NewProgram(newMetricsModel(name))

	go func() {
		for metrics := range metricsChan {
			p.Send(metrics)
		}
	}()

	return p, metricsChan
}
