package logdetail

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/search"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type tab int

const (
	tabRequestBody tab = iota
	tabResponseBody
)

type focusedPanel int

const (
	focusHeaderPanel focusedPanel = iota
	focusBodyPanel
)

const (
	typeDebounceDuration = 100 * time.Millisecond
	minTextViewWidth     = 50
	minAbsWidth          = 150
	minAbsHeight         = 15
	minHeight            = 10
)

type searchTickMsg time.Time

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case cursor.BlinkMsg:
		input := m.activeSearchInput()
		*input, cmd = input.Update(msg)
		return m, cmd

	case searchTickMsg:
		if m.searchActive && time.Since(m.lastKeyPress) >= typeDebounceDuration {
			matches := m.activeSearchMatches()
			*matches = search.FindMatches(
				m.activeContent(),
				m.activeSearchInput().Value(),
				m.activeLineOffsets(),
				m.normalCase,
			)
			if len(*matches) > 0 {
				m.resetCurrentMatch()
				m.jumpToCurrentMatch()
			}
		}
		return m, nil

	case messages.SetLogMsg:
		m.log = msg.Log
		m.headerSearchMatches = nil
		m.bodySearchMatches = nil
		m.headerCurrentMatch = 0
		m.bodyCurrentMatch = 0
		m.searchActive = false
		m.headerYOffset = 0
		m.headerXOffset = 0
		m.textYOffset = 0
		m.hexYOffset = 0
		m.xOffset = 0
		m.loadContent()
		m.calculateBodyHeight()
		m.calculateBodyWidth()
		m.calculateHeaderHeight()
		return m, nil

	case tea.WindowSizeMsg:
		m.absWidth = max(msg.Width, minAbsWidth)
		m.absHeight = max(msg.Height, minAbsHeight)
		m.headerViewWidth = (m.absWidth / 2)

		m.calculateBodyHeight()
		m.calculateBodyWidth()
		m.calculateHeaderHeight()

		m.loadContent()
		return m, nil

	case tea.KeyMsg:
		if m.searchActive {
			return m.handleSearchInput(msg)
		}

		switch {
		case key.Matches(msg, m.keys.DisplayText):
			m.displayText = !m.displayText
			m.calculateBodyHeight()
			return m, nil

		case key.Matches(msg, m.keys.DisplayHex):
			m.displayHex = !m.displayHex
			m.calculateBodyWidth()
			m.calculateBodyHeight()

			if m.wrapBody {
				m.visualLines = search.GetWrappedLines(m.stringContent, m.lineOffsets, m.viewWidth)
			}
			return m, nil

		case key.Matches(msg, m.keys.NormalCase):
			m.normalCase = !m.normalCase
			m.findMatches()
			m.calculateHeaderHeight()
			return m, nil

		case key.Matches(msg, m.keys.WrapText):
			if m.focusedPanel == focusHeaderPanel {
				var currentByteOffset int
				if m.wrapHeaders {
					if m.headerYOffset < len(m.headerVisualLines) {
						currentByteOffset = m.headerVisualLines[m.headerYOffset].StartOffset
					}
				} else {
					if m.headerYOffset < len(m.headerLineOffsets) {
						currentByteOffset = m.headerLineOffsets[m.headerYOffset] + m.headerXOffset
					}
				}

				m.wrapHeaders = !m.wrapHeaders

				if m.wrapHeaders {
					m.headerVisualLines = search.GetWrappedLines(m.headerContent, m.headerLineOffsets, m.headerViewWidth)
					m.headerYOffset = 0
					for i, vLine := range m.headerVisualLines {
						if currentByteOffset >= vLine.StartOffset && currentByteOffset < vLine.StartOffset+vLine.Length {
							m.headerYOffset = i
							break
						}
					}
					m.headerXOffset = 0
				} else {
					m.headerYOffset = 0
					for i, lineStart := range m.headerLineOffsets {
						next := len(m.headerContent)
						if i+1 < len(m.headerLineOffsets) {
							next = m.headerLineOffsets[i+1]
						}
						if currentByteOffset >= lineStart && currentByteOffset < next {
							m.headerYOffset = i
							m.headerXOffset = currentByteOffset - lineStart
							break
						}
					}
				}
			} else {
				var currentByteOffset int
				if m.wrapBody {
					if m.textYOffset < len(m.visualLines) {
						currentByteOffset = m.visualLines[m.textYOffset].StartOffset
					}
				} else {
					if m.textYOffset < len(m.lineOffsets) {
						currentByteOffset = m.lineOffsets[m.textYOffset] + m.xOffset
					}
				}

				m.wrapBody = !m.wrapBody

				if m.wrapBody {
					m.visualLines = search.GetWrappedLines(m.stringContent, m.lineOffsets, m.viewWidth)
					m.textYOffset = 0
					for i, vLine := range m.visualLines {
						if currentByteOffset >= vLine.StartOffset && currentByteOffset < vLine.StartOffset+vLine.Length {
							m.textYOffset = i
							break
						}
					}
					m.xOffset = 0
				} else {
					m.textYOffset = 0
					for i, lineStart := range m.lineOffsets {
						next := len(m.stringContent)
						if i+1 < len(m.lineOffsets) {
							next = m.lineOffsets[i+1]
						}
						if currentByteOffset >= lineStart && currentByteOffset < next {
							m.textYOffset = i
							m.xOffset = currentByteOffset - lineStart
							break
						}
					}
				}
			}
			m.clampOffsets()
			m.jumpToCurrentMatch()
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.calculateHeaderHeight()
			return m, nil

		case key.Matches(msg, m.keys.Search):
			m.searchActive = true
			input := m.activeSearchInput()
			input.Focus()
			m.calculateHeaderHeight()
			return m, nil

		case key.Matches(msg, m.keys.NextMatch):
			matches := m.activeSearchMatches()
			if len(*matches) > 0 {
				currentMatch := m.activeCurrentMatch()
				*currentMatch = (*currentMatch + 1) % len(*matches)
				m.jumpToCurrentMatch()
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevMatch):
			matches := m.activeSearchMatches()
			if len(*matches) > 0 {
				currentMatch := m.activeCurrentMatch()
				*currentMatch = (*currentMatch - 1 + len(*matches)) % len(*matches)
				m.jumpToCurrentMatch()
			}
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			m.activeTab = (m.activeTab + 1) % 2
			m.loadContent()
			return m, nil

		case key.Matches(msg, m.keys.Focus):
			m.focusedPanel = (m.focusedPanel + 1) % 2
			m.findMatches()
			m.calculateHeaderHeight()
			m.calculateHeaderHeight()
			return m, nil

		case key.Matches(msg, m.keys.Up):
			if m.focusedPanel == focusHeaderPanel {
				m.headerYOffset = max(0, m.headerYOffset-1)
			} else {
				m.textYOffset = max(0, m.textYOffset-1)
				m.hexYOffset = max(0, m.hexYOffset-1)
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.focusedPanel == focusHeaderPanel {
				totalRows := len(m.headerLineOffsets)
				if m.wrapHeaders {
					totalRows = len(m.headerVisualLines)
				}
				maxHeaderY := max(0, totalRows-m.headerViewHeight)
				m.headerYOffset = min(maxHeaderY, m.headerYOffset+1)
			} else {
				totalTextRows := len(m.lineOffsets)
				if m.wrapBody {
					totalTextRows = len(m.visualLines)
				}
				maxTextY := max(0, totalTextRows-m.viewHeight)
				maxHexY := max(0, m.totalHexRows-m.viewHeight)

				m.textYOffset = min(maxTextY, m.textYOffset+1)
				m.hexYOffset = min(maxHexY, m.hexYOffset+1)
			}
			return m, nil

		case key.Matches(msg, m.keys.GotoTop):
			if m.focusedPanel == focusHeaderPanel {
				m.headerYOffset = 0
			} else {
				m.textYOffset = 0
				m.hexYOffset = 0
			}
			return m, nil

		case key.Matches(msg, m.keys.GotoTopAlt):
			if time.Since(m.lastGPress) < 500*time.Millisecond {
				if m.focusedPanel == focusHeaderPanel {
					m.headerYOffset = 0
				} else {
					m.textYOffset = 0
					m.hexYOffset = 0
				}
			}
			m.lastGPress = time.Now()
			return m, nil

		case key.Matches(msg, m.keys.GotoBottom):
			if m.focusedPanel == focusHeaderPanel {
				totalRows := len(m.headerLineOffsets)
				if m.wrapHeaders {
					totalRows = len(m.headerVisualLines)
				}
				m.headerYOffset = max(0, totalRows-m.headerViewHeight)
			} else {
				totalTextRows := len(m.lineOffsets)
				if m.wrapBody {
					totalTextRows = len(m.visualLines)
				}
				m.textYOffset = max(0, totalTextRows-m.viewHeight)
				m.hexYOffset = max(0, m.totalHexRows-m.viewHeight)
			}
			return m, nil

		case key.Matches(msg, m.keys.GotoBottomAlt):
			if m.focusedPanel == focusHeaderPanel {
				totalRows := len(m.headerLineOffsets)
				if m.wrapHeaders {
					totalRows = len(m.headerVisualLines)
				}
				m.headerYOffset = max(0, totalRows-m.headerViewHeight)
			} else {
				totalTextRows := len(m.lineOffsets)
				if m.wrapBody {
					totalTextRows = len(m.visualLines)
				}
				m.textYOffset = max(0, totalTextRows-m.viewHeight)
				m.hexYOffset = max(0, m.totalHexRows-m.viewHeight)
			}
			return m, nil

		case key.Matches(msg, m.keys.GoToLeft):
			if m.focusedPanel == focusHeaderPanel {
				m.headerXOffset = 0
			} else {
				m.xOffset = 0
			}
			return m, nil

		case key.Matches(msg, m.keys.GoToRight):
			if m.focusedPanel == focusHeaderPanel {
				maxScroll := max(0, m.maxHeaderLineLength-(m.absWidth/2))
				m.headerXOffset = maxScroll
			} else {
				maxScroll := max(0, m.maxLineLength-m.viewWidth)
				m.xOffset = maxScroll
			}
			return m, nil

		case key.Matches(msg, m.keys.Left):
			if m.focusedPanel == focusHeaderPanel {
				m.headerXOffset = max(0, m.headerXOffset-1)
			} else {
				m.xOffset = max(0, m.xOffset-1)
			}
			return m, nil

		case key.Matches(msg, m.keys.Right):
			if m.focusedPanel == focusHeaderPanel {
				maxScroll := max(0, m.maxHeaderLineLength-(m.absWidth/2))
				m.headerXOffset = min(maxScroll, m.headerXOffset+1)
			} else {
				maxHorizScroll := max(0, m.maxLineLength-m.viewWidth)
				m.xOffset = min(maxHorizScroll, m.xOffset+1)
			}
			return m, nil
		}
	}

	return m, cmd
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.CancelSearch):
		input := m.activeSearchInput()
		if input.Value() == "" {
			m.searchActive = false
			input.Blur()
			m.calculateHeaderHeight()
			return m, nil
		}
		input.SetValue("")
		matches := m.activeSearchMatches()
		*matches = nil
		m.resetCurrentMatch()
		m.calculateHeaderHeight()
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		m.searchActive = false
		input := m.activeSearchInput()
		input.Blur()
		matches := m.activeSearchMatches()
		if len(*matches) > 0 {
			m.jumpToCurrentMatch()
		}
		m.calculateHeaderHeight()
		return m, nil
	}

	input := m.activeSearchInput()
	*input, cmd = input.Update(msg)
	m.lastKeyPress = time.Now()

	return m, tea.Batch(cmd, m.scheduleSearch())
}

func (m *Model) scheduleSearch() tea.Cmd {
	return tea.Tick(typeDebounceDuration, func(t time.Time) tea.Msg {
		return searchTickMsg(t)
	})
}

func (m *Model) calculateHeaderHeight() {
	m.helpHeight = 2
	if m.help.ShowAll {
		m.helpHeight = 9
	}

	matches := m.activeSearchMatches()
	if m.searchActive || len(*matches) > 0 {
		m.helpHeight += 1
	}

	height := m.absHeight - /*meta + header*/ 13 - m.helpHeight

	m.headerViewHeight = max(height, minHeight)
}

func (m *Model) resetCurrentMatch() {
	currentMatch := m.activeCurrentMatch()
	*currentMatch = 0
}

func (m *Model) calculateBodyHeight() {
	m.viewHeight = max(m.absHeight, minHeight) - 2

	if m.displayHex && m.displayText {
		m.viewHeight -= 3
	} else if m.displayHex || m.displayText {
		m.viewHeight -= 2
	}
}

func (m *Model) calculateBodyWidth() {
	if m.displayHex {
		hexColumnWidth := (styles.HexColumnSize * 3) - /*trimmed space at last hex*/ 1
		m.viewWidth = m.absWidth - m.headerViewWidth - /*column padding*/ 4 - hexColumnWidth
	} else {
		m.viewWidth = m.absWidth - m.headerViewWidth - /*column padding*/ 2
	}
}
