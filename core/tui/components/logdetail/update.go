package logdetail

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/search"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.SetLogMsg:
		m.log = msg.Log
		m.loadContent()

	case tea.WindowSizeMsg:
		m.viewWidth = 50
		m.viewHeight = 20

	case tea.KeyMsg:
		if m.searchActive {
			return m.handleSearchInput(msg)
		}

		switch {
		case key.Matches(msg, m.keys.NormalCase):
			m.normalCase = !m.normalCase
			m.loadContent()
			return m, nil

		case key.Matches(msg, m.keys.WrapText):
			var currentByteOffset int
			if m.wrapText {
				if m.textYOffset < len(m.visualLines) {
					currentByteOffset = m.visualLines[m.textYOffset].StartOffset
				}
			} else {
				if m.textYOffset < len(m.lineOffsets) {
					lineStart := m.lineOffsets[m.textYOffset]
					currentByteOffset = lineStart + m.xOffset
				}
			}

			m.wrapText = !m.wrapText

			if m.wrapText {
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
					nextLineStart := len(m.stringContent)
					if i+1 < len(m.lineOffsets) {
						nextLineStart = m.lineOffsets[i+1]
					}

					if currentByteOffset >= lineStart && currentByteOffset < nextLineStart {
						m.textYOffset = i
						m.xOffset = currentByteOffset - lineStart
						break
					}
				}
			}

			m.clampOffsets()
			m.jumpToCurrentMatch()
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keys.Search):
			m.searchActive = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case key.Matches(msg, m.keys.NextMatch):
			if len(m.searchMatches) > 0 {
				m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
				m.jumpToCurrentMatch()
			}

		case key.Matches(msg, m.keys.PrevMatch):
			if len(m.searchMatches) > 0 {
				m.currentMatch = (m.currentMatch - 1 + len(m.searchMatches)) % len(m.searchMatches)
				m.jumpToCurrentMatch()
			}

		case key.Matches(msg, m.keys.Tab):
			m.activeTab = (m.activeTab + 1) % 2
			m.loadContent()

		case key.Matches(msg, m.keys.Up):
			m.textYOffset = max(0, m.textYOffset-1)
			m.hexYOffset = max(0, m.hexYOffset-1)

		case key.Matches(msg, m.keys.Down):
			totalTextRows := len(m.lineOffsets)
			if m.wrapText {
				totalTextRows = len(m.visualLines)
			}

			maxTextY := max(0, totalTextRows-m.viewHeight)
			maxHexY := max(0, m.totalHexRows-m.viewHeight)

			m.textYOffset = min(maxTextY, m.textYOffset+1)
			m.hexYOffset = min(maxHexY, m.hexYOffset+1)

		case key.Matches(msg, m.keys.GotoTop):
			if time.Since(m.lastGPress) < 500*time.Millisecond {
				m.textYOffset = 0
				m.hexYOffset = 0
			}
			m.lastGPress = time.Now()

		case key.Matches(msg, m.keys.GotoBottom):
			totalTextRows := len(m.lineOffsets)
			if m.wrapText {
				totalTextRows = len(m.visualLines)
			}
			m.textYOffset = max(0, totalTextRows-m.viewHeight)
			m.hexYOffset = max(0, m.totalHexRows-m.viewHeight)

		case key.Matches(msg, m.keys.GoToLeft):
			m.xOffset = 0

		case key.Matches(msg, m.keys.GoToRight):
			maxScroll := max(0, m.maxLineLength-m.viewWidth)
			m.xOffset = maxScroll

		case key.Matches(msg, m.keys.Left):
			m.xOffset = max(0, m.xOffset-1)

		case key.Matches(msg, m.keys.Right):
			maxHorizScroll := max(0, m.maxLineLength-m.viewWidth)
			m.xOffset = min(maxHorizScroll, m.xOffset+1)
		}
	}

	return m, cmd
}
