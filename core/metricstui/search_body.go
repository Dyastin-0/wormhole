package metricstui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *metricsModel) findMatches(refresh bool) {
	if m.searchQuery == "" {
		m.searchMatches = nil
		m.currentMatch = 0
		if refresh {
			m.refreshViewportContent()
		}
		return
	}

	query := strings.ToLower(strings.ReplaceAll(m.searchQuery, "\\n", "\n"))
	lowerContent := strings.ToLower(m.stringContent)
	queryLen := len(query)

	matches := make([]*matchLocation, 0, len(m.stringContent)/50)

	offset := 0
	currentLineIdx := 0

	for {
		idx := strings.Index(lowerContent[offset:], query)
		if idx == -1 {
			break
		}

		actualStart := offset + idx
		actualEnd := actualStart + queryLen

		for currentLineIdx+1 < len(m.lineOffsets) && m.lineOffsets[currentLineIdx+1] <= actualStart {
			currentLineIdx++
		}

		matches = append(matches, &matchLocation{
			line:  currentLineIdx,
			start: actualStart,
			end:   actualEnd,
		})

		offset = actualStart + 1
	}

	m.searchMatches = matches
	if refresh {
		m.refreshViewportContent()
	}
}

func (m *metricsModel) highlightMatches(content string) string {
	if m.searchQuery == "" || len(m.searchMatches) == 0 {
		return content
	}

	startLine := m.viewport.YOffset
	endLine := min(len(m.lineOffsets)-1, startLine+m.viewport.Height)

	var result strings.Builder
	result.Grow(len(m.lineOffsets) + (m.viewport.Height * m.viewport.Width))

	for range startLine {
		result.WriteByte('\n')
	}

	for i := startLine; i <= endLine; i++ {
		lineStart := m.lineOffsets[i]
		lineEnd := len(content)
		if i+1 < len(m.lineOffsets) {
			lineEnd = m.lineOffsets[i+1] - 1
		}

		lineText := content[lineStart:lineEnd]
		result.WriteString(m.highlightLine(lineText, lineStart))
		result.WriteByte('\n')
	}

	remaining := (len(m.lineOffsets) - 1) - endLine
	for range remaining {
		result.WriteByte('\n')
	}

	return result.String()
}

func (m *metricsModel) highlightHexMatches() string {
	startRow := m.hexViewport.YOffset
	endRow := min(m.totalHexRows-1, startRow+m.hexViewport.Height+2)

	var result strings.Builder
	result.Grow(m.totalHexRows + (m.hexViewport.Height * 80))

	for range startRow {
		result.WriteByte('\n')
	}

	for i := startRow; i <= endRow; i++ {
		rowStartByte := i * hexColumnSize
		rowEndByte := min(rowStartByte+hexColumnSize, len(m.stringContent))

		rowBytes := m.stringContent[rowStartByte:rowEndByte]
		lineText := hexLine(rowBytes)

		if m.searchQuery != "" && len(m.searchMatches) > 0 {
			result.WriteString(m.highlightHexLine(lineText, rowStartByte))
		} else {
			result.WriteString(lineText)
		}
		result.WriteByte('\n')
	}

	remaining := (m.totalHexRows - 1) - endRow
	for range remaining {
		result.WriteByte('\n')
	}

	return result.String()
}

func (m *metricsModel) highlightLine(lineText string, lineStartOffset int) string {
	var result strings.Builder

	startIndex := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].end > lineStartOffset
	})

	runes := []rune(lineText)
	for i := range runes {
		globalIdx := lineStartOffset + i
		var activeStyle *lipgloss.Style

		for j := startIndex; j < len(m.searchMatches); j++ {
			match := m.searchMatches[j]
			if match.start > globalIdx {
				break
			}

			if globalIdx >= match.start && globalIdx < match.end {
				if j == m.currentMatch {
					activeStyle = &currentMatchStyle
					break
				}
				activeStyle = &searchHighlightStyle
			}
		}

		char := string(runes[i])
		if activeStyle != nil {
			result.WriteString(activeStyle.Render(char))
		} else {
			result.WriteString(char)
		}
	}
	return result.String()
}

func (m *metricsModel) highlightHexLine(lineText string, lineStartByte int) string {
	var result strings.Builder

	startIndex := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].end > lineStartByte
	})

	byteCount := (len(lineText) + 1) / 3

	for i := range byteCount {
		globalByteIdx := lineStartByte + i
		charIdx := i * 3

		var activeStyle *lipgloss.Style

		for j := startIndex; j < len(m.searchMatches); j++ {
			match := m.searchMatches[j]
			if match.start > globalByteIdx {
				break
			}
			if globalByteIdx >= match.start && globalByteIdx < match.end {
				if j == m.currentMatch {
					activeStyle = &currentMatchStyle
					break
				}
				activeStyle = &searchHighlightStyle
			}
		}

		hexPair := lineText[charIdx : charIdx+2]

		if activeStyle != nil {
			result.WriteString(activeStyle.Render(hexPair))
		} else {
			result.WriteString(hexPair)
		}

		if charIdx+2 < len(lineText) {
			result.WriteByte(' ')
		}
	}

	return result.String()
}

func (m *metricsModel) jumpToMatch() {
	if len(m.searchMatches) == 0 || m.currentMatch >= len(m.searchMatches) {
		return
	}
	match := m.searchMatches[m.currentMatch]

	targetY := max(0, match.line-(m.viewport.Height/2))
	m.viewport.SetYOffset(targetY)

	lineStart := m.lineOffsets[match.line]
	column := match.start - lineStart
	m.viewport.SetXOffset(max(0, column-(m.viewport.Width/2)))

	hexLine := match.start / hexColumnSize
	m.hexViewport.SetYOffset(max(0, hexLine-(m.hexViewport.Height/2)))

	m.refreshViewportContent()
}

func getLineOffsets(content string) []int {
	offsets := make([]int, 0, len(content)/40)
	offsets = append(offsets, 0)
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}
