package metricstui

import (
	"sort"
	"strings"
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

	query := strings.ReplaceAll(m.searchQuery, "\\n", "\n")
	query = strings.ToLower(query)

	content := m.stringContent
	queryLen := len(query)

	matches := make([]*matchLocation, 0, 1000)
	currentLineIdx := 0
	offset := 0

	lowerContent := strings.ToLower(content)

	for {
		idx := strings.Index(lowerContent[offset:], query)
		if idx == -1 {
			break
		}

		actualIdx := offset + idx

		for currentLineIdx+1 < len(m.lineOffsets) && m.lineOffsets[currentLineIdx+1] <= actualIdx {
			currentLineIdx++
		}

		matches = append(matches, &matchLocation{
			line:  currentLineIdx,
			start: actualIdx,
			end:   actualIdx + queryLen,
		})

		offset = actualIdx + queryLen
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
		lineText := m.formatSingleHexRow(rowBytes)

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

const hexChars = "0123456789ABCDEF"

func (m *metricsModel) formatSingleHexRow(data string) string {
	if len(data) == 0 {
		return ""
	}

	res := make([]byte, len(data)*3)
	for i := 0; i < len(data); i++ {
		res[i*3] = hexChars[data[i]>>4]
		res[i*3+1] = hexChars[data[i]&0x0f]
		res[i*3+2] = ' '
	}
	return string(res[:len(res)-1])
}

func (m *metricsModel) highlightLine(lineText string, lineStartOffset int) string {
	var result strings.Builder
	lastIdx := 0
	lineEndOffset := lineStartOffset + len(lineText)

	startIndex := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].end > lineStartOffset
	})

	for i := startIndex; i < len(m.searchMatches); i++ {
		match := m.searchMatches[i]

		if match.start >= lineEndOffset {
			break
		}

		relStart := max(0, match.start-lineStartOffset)
		relEnd := min(len(lineText), match.end-lineStartOffset)

		if relStart < lastIdx {
			continue
		}

		result.WriteString(lineText[lastIdx:relStart])

		target := lineText[relStart:relEnd]
		if i == m.currentMatch {
			result.WriteString(currentMatchStyle.Render(target))
		} else {
			result.WriteString(searchHighlightStyle.Render(target))
		}

		lastIdx = relEnd
	}

	result.WriteString(lineText[lastIdx:])
	return result.String()
}

func (m *metricsModel) highlightHexLine(lineText string, lineStartByte int) string {
	var result strings.Builder
	lastCharIdx := 0
	lineEndByte := lineStartByte + hexColumnSize

	startIndex := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].end > lineStartByte
	})

	for i := startIndex; i < len(m.searchMatches); i++ {
		match := m.searchMatches[i]
		if match.start >= lineEndByte {
			break
		}

		rowMatchStart := max(lineStartByte, match.start)
		rowMatchEnd := min(lineEndByte, match.end)

		relStartChar := (rowMatchStart - lineStartByte) * 3
		relEndChar := (rowMatchEnd - lineStartByte) * 3

		if relEndChar > len(lineText) {
			relEndChar = len(lineText)
		} else if relEndChar > 0 && lineText[relEndChar-1] == ' ' {
			relEndChar--
		}

		result.WriteString(lineText[lastCharIdx:relStartChar])

		target := lineText[relStartChar:relEndChar]
		if i == m.currentMatch {
			result.WriteString(currentMatchStyle.Render(target))
		} else {
			result.WriteString(searchHighlightStyle.Render(target))
		}

		lastCharIdx = relEndChar
	}

	result.WriteString(lineText[lastCharIdx:])
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
