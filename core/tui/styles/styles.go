// Package styles provides styles across the TUI.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

const (
	TimeWidth     = 8
	MethodWidth   = 7
	StatusWidth   = 3
	PathWidth     = 40
	SizeWidth     = 10
	DurationWidth = 10
	LabelWidth    = 12

	HeaderKeyWidth   = 30
	HeaderValueWidth = 50

	HexColumnSize = 8
)

var (
	Subtle     = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#808080"}
	Primary    = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}
	SelectedBG = lipgloss.AdaptiveColor{Light: "#E6F3FF", Dark: "#1A3A52"}
	Highlight  = lipgloss.Color("#F07178")
)

var (
	SearchHighlight = lipgloss.NewStyle().
			Background(lipgloss.Color("226")).
			Foreground(lipgloss.Color("#1a1a1a"))

	CurrentMatch = lipgloss.NewStyle().
			Background(lipgloss.Color("208")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	Label = lipgloss.NewStyle().
		Foreground(Subtle).
		Width(20).
		Faint(true).
		Align(lipgloss.Left)

	Text = lipgloss.NewStyle().
		Foreground(Primary)

	Value = lipgloss.NewStyle().
		Foreground(Primary).
		Width(12).
		Align(lipgloss.Right)

	Rate = lipgloss.NewStyle().
		Foreground(Subtle).
		Width(12).
		Align(lipgloss.Right)

	HelpKey = lipgloss.NewStyle().
		Foreground(Highlight)

	Footer = lipgloss.NewStyle().
		Foreground(Subtle).
		Faint(true)

	DetailLabel = lipgloss.NewStyle().
			Foreground(Subtle).
			Bold(true).
			Faint(true)
)

var (
	LogTime = lipgloss.NewStyle().
		Foreground(Primary).
		Width(TimeWidth).
		Align(lipgloss.Left)

	LogMethod = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Width(MethodWidth).
			Align(lipgloss.Left)

	LogPath = lipgloss.NewStyle().
		Foreground(Primary).
		Width(PathWidth).
		Align(lipgloss.Left)

	LogSize = lipgloss.NewStyle().
		Foreground(Primary).
		Width(SizeWidth).
		Align(lipgloss.Left)

	LogDuration = lipgloss.NewStyle().
			Foreground(Primary).
			Width(DurationWidth).
			Align(lipgloss.Right)
)

var (
	Status1xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00afff")).
			Width(StatusWidth).
			Bold(true)
	Status2xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00d700")).
			Width(StatusWidth).
			Bold(true)
	Status3xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffaf00")).
			Width(StatusWidth).
			Bold(true)
	Status4xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff3333")).
			Width(StatusWidth).
			Bold(true)
	Status5xx = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			Width(StatusWidth).
			Bold(true).
			Italic(true)
)
