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

	absWidth  int
	absHeight int

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

	bodySearchMatches   []search.Match
	headerSearchMatches []search.Match
	bodySearchInput     textinput.Model
	headerSearchInput   textinput.Model

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

	bodyInput := textinput.New()
	bodyInput.TextStyle = styles.Text
	bodyInput.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Highlight)
	bodyInput.CharLimit = 100
	bodyInput.PromptStyle = styles.Text
	bodyInput.Prompt = "> "

	headerInput := textinput.New()
	headerInput.TextStyle = styles.Text
	headerInput.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Highlight)
	headerInput.CharLimit = 100
	headerInput.PromptStyle = styles.Text
	headerInput.Prompt = "> "

	return Model{
		activeTab:         tabResponseBody,
		bodySearchInput:   bodyInput,
		headerSearchInput: headerInput,
		keys:              DefaultKeyMap(),
		help:              h,
		helpHeight:        2,
		displayHex:        true,
		displayText:       true,
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
	switch m.focusedPanel {
	case focusHeaderPanel:
		if m.headerSearchInput.Value() == "" {
			m.headerSearchMatches = nil
			m.headerCurrentMatch = 0
			return
		}

		m.headerSearchMatches = search.FindMatches(m.headerContent, m.headerSearchInput.Value(), m.headerLineOffsets, m.normalCase)

		if len(m.headerSearchMatches) > 0 {
			if m.headerCurrentMatch >= len(m.headerSearchMatches) {
				m.headerCurrentMatch = 0
			}
			m.jumpToCurrentMatch()
		} else {
			m.headerCurrentMatch = 0
		}

	case focusBodyPanel:
		if m.bodySearchInput.Value() == "" {
			m.bodySearchMatches = nil
			m.bodyCurrentMatch = 0
			return
		}

		m.bodySearchMatches = search.FindMatches(m.stringContent, m.bodySearchInput.Value(), m.lineOffsets, m.normalCase)

		if len(m.bodySearchMatches) > 0 {
			if m.bodyCurrentMatch >= len(m.bodySearchMatches) {
				m.bodyCurrentMatch = 0
			}
			m.jumpToCurrentMatch()
		} else {
			m.bodyCurrentMatch = 0
		}
	}
}

func (m *Model) jumpToCurrentMatch() {
	switch m.focusedPanel {
	case focusHeaderPanel:
		if len(m.headerSearchMatches) == 0 || m.headerCurrentMatch >= len(m.headerSearchMatches) {
			return
		}
		m.headerYOffset, _, m.headerXOffset = search.JumpToMatch(
			m.headerSearchMatches[m.headerCurrentMatch],
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
		if len(m.bodySearchMatches) == 0 || m.bodyCurrentMatch >= len(m.bodySearchMatches) {
			return
		}
		m.textYOffset, m.hexYOffset, m.xOffset = search.JumpToMatch(
			m.bodySearchMatches[m.bodyCurrentMatch],
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

// Helper methods
func (m *Model) activeSearchInput() *textinput.Model {
	if m.focusedPanel == focusHeaderPanel {
		return &m.headerSearchInput
	}
	return &m.bodySearchInput
}

func (m *Model) activeSearchMatches() *[]search.Match {
	if m.focusedPanel == focusHeaderPanel {
		return &m.headerSearchMatches
	}
	return &m.bodySearchMatches
}

func (m *Model) activeCurrentMatch() *int {
	if m.focusedPanel == focusHeaderPanel {
		return &m.headerCurrentMatch
	}
	return &m.bodyCurrentMatch
}

func (m *Model) activeContent() string {
	if m.focusedPanel == focusHeaderPanel {
		return m.headerContent
	}
	return m.stringContent
}

func (m *Model) activeLineOffsets() []int {
	if m.focusedPanel == focusHeaderPanel {
		return m.headerLineOffsets
	}
	return m.lineOffsets
}
