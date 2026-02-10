// Package logdetail implements the log detail component.
package logdetail

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/search"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	log          *messages.HTTPLogMsg
	activeTab    tab
	focusedPanel focusedPanel

	absWidth int

	headerContent       string
	headerLineOffsets   []int
	headerVisualLines   []search.VisualLine
	headerYOffset       int
	headerXOffset       int
	maxHeaderLineLength int
	headerViewHeight    int
	headerViewWidth     int

	stringContent string
	lineOffsets   []int
	visualLines   []search.VisualLine
	maxLineLength int
	viewWidth     int
	viewHeight    int
	textYOffset   int
	hexYOffset    int
	xOffset       int
	totalHexRows  int

	wrapBody     bool
	wrapHeaders  bool
	normalCase   bool
	searchActive bool

	searchMatches []search.Match
	searchInput   textinput.Model

	bodyCurrentMatch   int
	headerCurrentMatch int

	displayHex  bool
	displayText bool

	lastKeyPress time.Time
	lastGPress   time.Time
	keys         KeyMap
	help         help.Model
	helpHeight   int
}

func New() Model {
	h := help.New()
	h.ShortSeparator = " • "
	h.Styles.ShortSeparator = styles.Label.Width(0)
	h.Styles.ShortKey = styles.HelpKey
	h.Styles.FullKey = styles.HelpKey
	h.Styles.ShortDesc = styles.Footer.Faint(true)
	h.Styles.FullDesc = styles.Footer.Faint(true)
	h.Styles.ShortSeparator = styles.Footer.Faint(true)

	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Highlight)
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Prompt = ""

	return Model{
		activeTab:   tabResponseBody,
		searchInput: ti,
		keys:        DefaultKeyMap(),
		help:        h,
		displayHex:  true,
		displayText: true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) IsSearching() bool {
	return m.searchActive
}

func (m *Model) clampOffsets() {
	totalRows := len(m.lineOffsets)
	if m.wrapBody {
		totalRows = len(m.visualLines)
		m.xOffset = 0
	}

	m.textYOffset = max(0, min(m.textYOffset, max(0, totalRows-m.viewHeight)))

	if !m.wrapBody {
		maxScrollX := max(0, m.maxLineLength-m.viewWidth)
		m.xOffset = max(0, min(m.xOffset, maxScrollX))
	}

	m.hexYOffset = max(0, min(m.hexYOffset, max(0, m.totalHexRows-m.viewHeight)))
}

func (m *Model) findMatches() {
	if m.searchInput.Value() == "" {
		m.searchMatches = nil
		m.bodyCurrentMatch = 0
		return
	}

	switch m.focusedPanel {
	case focusHeaderPanel:
		m.searchMatches = search.FindMatches(m.headerContent, m.searchInput.Value(), m.headerLineOffsets, m.normalCase)
	case focusBodyPanel:
		m.searchMatches = search.FindMatches(m.stringContent, m.searchInput.Value(), m.lineOffsets, m.normalCase)
	}

	if len(m.searchMatches) > 0 {
		if m.currentMatch() >= len(m.searchMatches) {
			m.resetCurrentMatch()
		}
		m.jumpToCurrentMatch()
	} else {
		m.resetCurrentMatch()
	}
}

func (m *Model) jumpToCurrentMatch() {
	if len(m.searchMatches) == 0 || m.bodyCurrentMatch >= len(m.searchMatches) || m.headerCurrentMatch > len(m.searchMatches) {
		return
	}

	switch m.focusedPanel {
	case focusHeaderPanel:
		m.headerYOffset, _, m.headerXOffset = search.JumpToMatch(
			m.searchMatches[m.headerCurrentMatch],
			len(m.headerContent),
			m.headerLineOffsets,
			m.headerVisualLines,
			m.wrapHeaders,
			m.headerViewHeight,
			m.headerViewWidth,
			styles.HexColumnSize,
			m.maxHeaderLineLength,
		)
	case focusBodyPanel:
		m.textYOffset, m.hexYOffset, m.xOffset = search.JumpToMatch(
			m.searchMatches[m.bodyCurrentMatch],
			len(m.stringContent),
			m.lineOffsets,
			m.visualLines,
			m.wrapBody,
			m.viewHeight,
			m.viewWidth,
			styles.HexColumnSize,
			m.maxLineLength,
		)
	}
}
