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
	log       *messages.HTTPLogMsg
	activeTab messages.Tab

	stringContent string
	lineOffsets   []int
	maxLineLength int
	totalHexRows  int

	absWidth    int
	viewWidth   int
	viewHeight  int
	textYOffset int
	hexYOffset  int
	xOffset     int

	wrapText     bool
	normalCase   bool
	searchActive bool
	displayHex   bool
	displayText  bool

	searchInput   textinput.Model
	searchMatches []search.Match
	visualLines   []search.VisualLine
	currentMatch  int

	lastKeyPress time.Time
	lastGPress   time.Time
	keys         KeyMap
	help         help.Model
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
		activeTab:   messages.TabResponseBody,
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
	if m.wrapText {
		totalRows = len(m.visualLines)
		m.xOffset = 0
	}

	m.textYOffset = max(0, min(m.textYOffset, max(0, totalRows-m.viewHeight)))

	if !m.wrapText {
		maxScrollX := max(0, m.maxLineLength-m.viewWidth)
		m.xOffset = max(0, min(m.xOffset, maxScrollX))
	}

	m.hexYOffset = max(0, min(m.hexYOffset, max(0, m.totalHexRows-m.viewHeight)))
}

func (m *Model) findMatches() {
	if m.searchInput.Value() != "" {
		m.searchMatches = search.FindMatches(m.stringContent, m.searchInput.Value(), m.lineOffsets, m.normalCase)
		if len(m.searchMatches) > 0 {
			m.currentMatch = 0
		}
	}
}

func (m *Model) jumpToCurrentMatch() {
	if len(m.searchMatches) == 0 || m.currentMatch >= len(m.searchMatches) {
		return
	}

	m.textYOffset, m.hexYOffset, m.xOffset = search.JumpToMatch(
		m.searchMatches[m.currentMatch],
		len(m.stringContent),
		m.lineOffsets,
		m.visualLines,
		m.wrapText,
		m.viewHeight,
		m.viewWidth,
		styles.HexColumnSize,
		m.maxLineLength,
	)
}
