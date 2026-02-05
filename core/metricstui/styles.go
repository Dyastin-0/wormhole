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
	viewportWidth = 50

	headerHeight = 12
	footerHeight = 3
)

var (
	subtle  = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#808080"}
	primary = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}

	selectedBG = lipgloss.AdaptiveColor{Light: "#E6F3FF", Dark: "#1A3A52"}
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
