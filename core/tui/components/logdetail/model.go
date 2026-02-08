// Package logdetail implements the log detail component.
package logdetail

import (
	"fmt"
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/formatters"
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/search"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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

	viewWidth   int
	viewHeight  int
	textYOffset int
	hexYOffset  int
	xOffset     int

	wrapText      bool
	normalCase    bool
	searchActive  bool
	searchInput   textinput.Model
	searchMatches []search.Match
	visualLines   []search.VisualLine
	currentMatch  int

	lastGPress time.Time
	keys       KeyMap
	help       help.Model
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
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Prompt = ""

	return Model{
		activeTab:   messages.TabResponseBody,
		searchInput: ti,
		keys:        DefaultKeyMap(),
		help:        h,
		viewWidth:   50,
		viewHeight:  20,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		if m.searchInput.Value() == "" {
			m.searchActive = false
			m.searchInput.Blur()
			return m, nil
		}
		m.searchInput.SetValue("")
		m.searchMatches = nil
		m.currentMatch = 0
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		m.searchActive = false
		m.searchInput.Blur()
		if len(m.searchMatches) > 0 {
			m.currentMatch = 0
			m.jumpToCurrentMatch()
		}
		return m, nil
	}

	m.searchInput, cmd = m.searchInput.Update(msg)

	if m.searchInput.Value() != "" {
		m.searchMatches = search.FindMatches(m.stringContent, m.searchInput.Value(), m.lineOffsets, m.normalCase)
		if len(m.searchMatches) > 0 {
			m.currentMatch = 0
		}
	} else {
		m.searchMatches = nil
		m.currentMatch = 0
	}

	return m, cmd
}

func (m Model) renderMetadata(title string) string {
	statusValue := formatters.FormatStatusCode(m.log.Response.StatusCode, false)

	metaLines := []string{
		title,
		"",
		m.formatDetailLine(
			"Timestamp",
			styles.LogTime.Width(styles.TimeWidth+11).Render(
				time.Unix(m.log.Timestamp, 0).Format("2006-01-02 15:04:05"),
			),
		),
		m.formatDetailLine(
			"Method",
			styles.LogMethod.Render(m.log.Request.Method),
		),
		m.formatDetailLine(
			"Path",
			styles.LogPath.Render(m.log.Request.URL.Path),
		),
		m.formatDetailLine(
			"Status",
			statusValue,
		),
		m.formatDetailLine(
			"Size",
			styles.LogSize.Render(formatters.FormatBytes(uint64(m.log.Response.Size))),
		),
		m.formatDetailLine(
			"Duration",
			styles.LogDuration.Align(lipgloss.Left).Render(
				fmt.Sprintf("%.2f ms", float64(m.log.Duration)/1000.0),
			),
		),
		"",
	}

	return lipgloss.JoinVertical(lipgloss.Left, metaLines...)
}

func (m Model) renderFooter() string {
	m.keys.Search.SetEnabled(!m.searchActive)
	helpView := m.help.View(m.keys)

	var statusLine string
	if m.searchActive {
		matchInfo := ""
		if len(m.searchMatches) > 0 {
			matchInfo = fmt.Sprintf(" (%d/%d)", m.currentMatch+1, len(m.searchMatches))
		}
		searchStr := styles.HelpKey.Render("Search")
		inputStr := styles.Value.Width(0).Render(fmt.Sprintf(" /%s%s", m.searchInput.Value(), matchInfo))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, searchStr, inputStr)
	} else if len(m.searchMatches) > 0 {
		matchStr := styles.HelpKey.Render("Match")
		statusLine = styles.Value.Width(0).Render(fmt.Sprintf(" %d/%d", m.currentMatch+1, len(m.searchMatches)))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, matchStr, statusLine)
	}

	if statusLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			statusLine,
			helpView,
		)
	}

	return helpView
}

func (m Model) renderBodyColumn() string {
	var textView, hexView string

	if m.wrapText {
		textView = search.HighlightWrappedMatches(
			m.stringContent,
			m.visualLines,
			m.searchMatches,
			m.currentMatch,
			m.textYOffset,
			m.viewHeight,
			m.viewWidth,
		)
	} else {
		textView = search.HighlightMatches(
			m.stringContent,
			m.lineOffsets,
			m.searchMatches,
			m.currentMatch,
			m.textYOffset,
			m.viewHeight,
			m.viewWidth,
			m.xOffset,
		)
	}

	hexView = search.HighlightHexMatches(
		m.stringContent,
		m.searchMatches,
		m.currentMatch,
		m.hexYOffset,
		m.viewHeight,
		styles.HexColumnSize,
	)

	text := styles.Text.Width(m.viewWidth).MaxWidth(m.viewWidth).Render(textView)
	hex := styles.Text.Render(hexView)
	viewports := lipgloss.JoinHorizontal(lipgloss.Top, text, "  ", hex)

	totalRows := len(m.lineOffsets)
	if m.wrapText {
		totalRows = len(m.visualLines)
	}

	currentRowEnd := min(totalRows, m.textYOffset+m.viewHeight)

	textYInfo := lipgloss.JoinHorizontal(lipgloss.Left,
		styles.HelpKey.Render("Text rows"),
		styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", m.textYOffset+1, currentRowEnd, totalRows)),
	)

	var textXInfo string
	if !m.wrapText {
		rightCol := min(m.maxLineLength, m.xOffset+m.viewWidth)
		textXInfo = lipgloss.JoinHorizontal(lipgloss.Left,
			styles.HelpKey.Render("Text cols"),
			styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", m.xOffset+1, rightCol, m.maxLineLength)),
		)
	}

	bottomHex := min(m.totalHexRows, m.hexYOffset+m.viewHeight)
	hexYInfo := lipgloss.JoinHorizontal(lipgloss.Left,
		styles.HelpKey.Render("Hex rows"),
		styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", m.hexYOffset+1, bottomHex, m.totalHexRows)),
	)

	sep := m.help.Styles.ShortSeparator.Render(m.help.ShortSeparator)
	textRow := textYInfo
	if textXInfo != "" {
		textRow = lipgloss.JoinHorizontal(lipgloss.Left, textYInfo, sep, textXInfo)
	}

	scrollInfo := lipgloss.JoinVertical(lipgloss.Left, textRow, hexYInfo)
	footerRow := lipgloss.NewStyle().MarginTop(1).Render(scrollInfo)

	return lipgloss.JoinVertical(lipgloss.Left,
		styles.Value.Width(0).Render(fmt.Sprintf("Body %s", formatters.FormatBytes(uint64(m.getBodySize())))),
		"",
		viewports,
		footerRow,
	)
}

func (m Model) formatDetailLine(label, styledValue string) string {
	l := styles.DetailLabel.Render(label)
	leftAlignedLabel := lipgloss.PlaceHorizontal(styles.LabelWidth-1, lipgloss.Left, l)
	fullLabel := fmt.Sprintf("%s:", leftAlignedLabel)
	return lipgloss.JoinHorizontal(lipgloss.Left, fullLabel, " ", styledValue)
}

func (m *Model) loadContent() {
	if m.log == nil {
		return
	}

	var content string
	if m.activeTab == messages.TabRequestBody {
		content = string(m.log.RequestBody)
	} else {
		content = string(m.log.ResponseBody)
	}

	m.stringContent = content
	m.lineOffsets, m.maxLineLength = search.GetLineOffsets(m.stringContent)
	m.totalHexRows = (len(m.stringContent) + styles.HexColumnSize - 1) / styles.HexColumnSize

	if m.wrapText {
		m.visualLines = search.GetWrappedLines(m.stringContent, m.lineOffsets, m.viewWidth)
	} else {
		m.visualLines = nil
	}

	m.textYOffset = 0
	m.hexYOffset = 0
	m.xOffset = 0

	m.findMatches()
}

func (m Model) getBodySize() int {
	if m.log == nil {
		return 0
	}
	if m.activeTab == messages.TabResponseBody {
		return len(m.log.ResponseBody)
	}
	return len(m.log.RequestBody)
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
