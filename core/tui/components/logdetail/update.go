package logdetail

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/search"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
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
	minHeight            = 10
)

type searchTickMsg time.Time

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case searchTickMsg:
		if m.searchActive && time.Since(m.lastKeyPress) >= typeDebounceDuration {
			if m.focusedPanel == focusHeaderPanel {
				m.searchMatches = search.FindMatches(m.headerContent, m.searchInput.Value(), m.headerLineOffsets, m.normalCase)
			} else {
				m.searchMatches = search.FindMatches(m.stringContent, m.searchInput.Value(), m.lineOffsets, m.normalCase)
			}
			if len(m.searchMatches) > 0 {
				m.resetCurrentMatch()
				m.jumpToCurrentMatch()
			}
		}
		return m, nil

	case messages.SetLogMsg:
		m.log = msg.Log
		m.loadContent()

	case tea.WindowSizeMsg:
		m.absWidth = max(msg.Width, minAbsWidth)
		windowHeight := max(msg.Height, minHeight)

		m.viewHeight = windowHeight - /*body header and footer*/ 5

		m.headerViewWidth = (m.absWidth / 2)
		m.calculateHeaderHeight()

		if m.displayHex {
			hexColumnWidth := (styles.HexColumnSize * 3) - /*trimmed space at last hex*/ 1
			m.viewWidth = m.absWidth - m.headerViewWidth - /*column padding*/ 4 - hexColumnWidth
		} else {
			m.viewWidth = m.absWidth - m.headerViewWidth - /*column padding*/ 2
		}
		m.loadContent()
		return m, nil

	case tea.KeyMsg:
		if m.searchActive {
			return m.handleSearchInput(msg)
		}

		switch {
		case key.Matches(msg, m.keys.DisplayText):
			m.displayText = !m.displayText
			return m, nil

		case key.Matches(msg, m.keys.DisplayHex):
			m.displayHex = !m.displayHex

			if m.displayHex {
				hexColumnWidth := (styles.HexColumnSize * 3) - /*trimmed space at last hex*/ 1
				m.viewWidth = m.absWidth - m.headerViewWidth - /*column padding*/ 4 - hexColumnWidth
			} else {
				m.viewWidth = m.absWidth - m.headerViewWidth - /*column padding*/ 2
			}

			if m.wrapBody {
				m.visualLines = search.GetWrappedLines(m.stringContent, m.lineOffsets, m.viewWidth)
			}
			return m, nil

		case key.Matches(msg, m.keys.NormalCase):
			m.normalCase = !m.normalCase
			m.findMatches()
			m.calculateHeaderHeight()

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

		case key.Matches(msg, m.keys.Search):
			m.searchActive = true
			m.searchInput.Focus()
			m.calculateHeaderHeight()
			return m, textinput.Blink

		case key.Matches(msg, m.keys.NextMatch):
			if len(m.searchMatches) > 0 {
				switch m.focusedPanel {
				case focusHeaderPanel:
					m.headerCurrentMatch = (m.headerCurrentMatch + 1) % len(m.searchMatches)
				case focusBodyPanel:
					m.bodyCurrentMatch = (m.bodyCurrentMatch + 1) % len(m.searchMatches)
				}
				m.jumpToCurrentMatch()
			}

		case key.Matches(msg, m.keys.PrevMatch):
			if len(m.searchMatches) > 0 {
				switch m.focusedPanel {
				case focusHeaderPanel:
					m.headerCurrentMatch = (m.headerCurrentMatch - 1 + len(m.searchMatches)) % len(m.searchMatches)
				case focusBodyPanel:
					m.bodyCurrentMatch = (m.bodyCurrentMatch - 1 + len(m.searchMatches)) % len(m.searchMatches)
				}
				m.jumpToCurrentMatch()
			}

		case key.Matches(msg, m.keys.Tab):
			m.activeTab = (m.activeTab + 1) % 2
			m.loadContent()

		case key.Matches(msg, m.keys.Focus):
			m.focusedPanel = (m.focusedPanel + 1) % 2
			m.findMatches()
			m.calculateHeaderHeight()

		case key.Matches(msg, m.keys.Up):
			if m.focusedPanel == focusHeaderPanel {
				m.headerYOffset = max(0, m.headerYOffset-1)
			} else {

				m.textYOffset = max(0, m.textYOffset-1)
				m.hexYOffset = max(0, m.hexYOffset-1)
			}

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

		case key.Matches(msg, m.keys.GotoTop):
			if m.focusedPanel == focusHeaderPanel {
				m.headerYOffset = 0
			} else {
				m.textYOffset = 0
				m.hexYOffset = 0
			}

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

		case key.Matches(msg, m.keys.GoToLeft):
			if m.focusedPanel == focusHeaderPanel {
				m.headerXOffset = 0
			} else {
				m.xOffset = 0
			}

		case key.Matches(msg, m.keys.GoToRight):
			if m.focusedPanel == focusHeaderPanel {
				maxScroll := max(0, m.maxHeaderLineLength-(m.absWidth/2))
				m.headerXOffset = maxScroll
			} else {
				maxScroll := max(0, m.maxLineLength-m.viewWidth)
				m.xOffset = maxScroll
			}
		case key.Matches(msg, m.keys.Left):
			if m.focusedPanel == focusHeaderPanel {
				m.headerXOffset = max(0, m.headerXOffset-1)
			} else {
				m.xOffset = max(0, m.xOffset-1)
			}

		case key.Matches(msg, m.keys.Right):
			if m.focusedPanel == focusHeaderPanel {
				maxScroll := max(0, m.maxHeaderLineLength-(m.absWidth/2))
				m.headerXOffset = min(maxScroll, m.headerXOffset+1)
			} else {
				maxHorizScroll := max(0, m.maxLineLength-m.viewWidth)
				m.xOffset = min(maxHorizScroll, m.xOffset+1)
			}
		}
	}

	return m, cmd
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.CancelSearch):
		if m.searchInput.Value() == "" {
			m.searchActive = false
			m.searchInput.Blur()
			m.helpHeight -= 1
			m.headerViewHeight = m.viewHeight - /*metadata height*/ 10 - m.helpHeight
			return m, nil
		}
		m.searchInput.SetValue("")
		m.searchMatches = nil
		m.resetCurrentMatch()
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		m.searchActive = false
		m.searchInput.Blur()
		m.helpHeight -= 1
		if len(m.searchMatches) > 0 {
			m.jumpToCurrentMatch()
			m.helpHeight += 1
		}
		m.headerViewHeight = m.viewHeight - /*metadata height*/ 10 - m.helpHeight
		return m, nil
	}

	m.searchInput, cmd = m.searchInput.Update(msg)
	m.lastKeyPress = time.Now()

	return m, tea.Batch(cmd, m.scheduleSearch())
}

func (m *Model) scheduleSearch() tea.Cmd {
	return tea.Tick(typeDebounceDuration, func(t time.Time) tea.Msg {
		return searchTickMsg(t)
	})
}

func (m *Model) calculateHeaderHeight() {
	m.helpHeight = 0
	if m.help.ShowAll {
		m.helpHeight = 7
	}

	if m.searchActive || len(m.searchMatches) > 0 {
		m.helpHeight += 1
	}

	m.headerViewHeight = max(m.viewHeight-10-m.helpHeight, 0)
}

func (m *Model) resetCurrentMatch() {
	switch m.focusedPanel {
	case focusHeaderPanel:
		m.headerCurrentMatch = 0
	case focusBodyPanel:
		m.bodyCurrentMatch = 0
	}
}
