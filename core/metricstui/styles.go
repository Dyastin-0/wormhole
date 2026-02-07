package metricstui

import (
	"github.com/charmbracelet/lipgloss"
)

const (
	timeWidth     = 8
	methodWidth   = 7
	statusWidth   = 3
	pathWidth     = 40
	sizeWidth     = 10
	durationWidth = 10
	labelWidth    = 12

	headerKeyWidth   = 30
	headerValueWidth = 50

	hexColumnSize = 8
)

var (
	subtle       = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#808080"}
	primary      = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}
	highlight    = lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#F07178"}
	selectedBG   = lipgloss.AdaptiveColor{Light: "#E6F3FF", Dark: "#1A3A52"}
	helpKeyStyle = lipgloss.NewStyle().Foreground(highlight)
)

var (
	searchHighlightStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("226")).
				Foreground(lipgloss.Color("#1a1a1a"))

	currentMatchStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("208")).
				Foreground(lipgloss.Color("#000000")).
				Bold(true)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary)

	labelStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Width(20).
			Faint(true).
			Align(lipgloss.Left)

	textStyle = lipgloss.NewStyle().
			Foreground(primary)

	valueStyle = lipgloss.NewStyle().
			Foreground(primary).
			Width(12).
			Align(lipgloss.Right)

	rateStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Width(12).
			Align(lipgloss.Right)

	logTimeStyle = lipgloss.NewStyle().
			Foreground(primary).
			Width(timeWidth).
			Align(lipgloss.Left)

	logMethodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Width(methodWidth).
			Align(lipgloss.Left)

	logStatus1xxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Width(statusWidth).
				Bold(true)

	logStatus2xxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Width(statusWidth).
				Bold(true)

	logStatus3xxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Width(statusWidth).
				Bold(true)

	logStatus4xxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("208")).
				Width(statusWidth).
				Bold(true)

	logStatus5xxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Width(statusWidth).
				Bold(true).
				Italic(true)

	logPathStyle = lipgloss.NewStyle().
			Foreground(primary).
			Width(pathWidth).
			Align(lipgloss.Left)

	logSizeStyle = lipgloss.NewStyle().
			Foreground(primary).
			Width(sizeWidth).
			Align(lipgloss.Left)

	logDurationStyle = lipgloss.NewStyle().
				Foreground(primary).
				Width(durationWidth).
				Align(lipgloss.Right)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(subtle).
				Bold(true).
				Faint(true)

	footerStyle = lipgloss.NewStyle().Foreground(subtle).Faint(true)
)
