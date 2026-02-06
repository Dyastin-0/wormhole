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

	query := strings.ToLower(strings.ReplaceAll(m.searchQuery, "\\n", "\n"))
	queryLen := len(query)

	matches := m.findMatchesBMH(query, queryLen)

	m.searchMatches = matches
	if refresh {
		m.refreshViewportContent()
	}
}

func (m *metricsModel) findMatchesBMH(query string, queryLen int) []*matchLocation {
	if queryLen == 0 {
		return nil
	}

	lowerContent := strings.ToLower(m.stringContent)
	contentLen := len(lowerContent)

	matches := make([]*matchLocation, 0, min(1000, contentLen/100))

	badChar := make(map[byte]int)

	for i := range queryLen {
		if i < queryLen-1 {
			badChar[query[i]] = queryLen - 1 - i
		}
	}

	findLine := func(pos int) int {
		return sort.Search(len(m.lineOffsets), func(i int) bool {
			return m.lineOffsets[i] > pos
		}) - 1
	}

	pos := 0
	for pos <= contentLen-queryLen {
		j := queryLen - 1
		for j >= 0 && lowerContent[pos+j] == query[j] {
			j--
		}

		if j < 0 {
			actualStart := pos
			actualEnd := pos + queryLen
			lineIdx := findLine(actualStart)

			matches = append(matches, &matchLocation{
				line:  lineIdx,
				start: actualStart,
				end:   actualEnd,
			})
			pos += 1
		} else {
			shift, ok := badChar[lowerContent[pos+queryLen-1]]
			if !ok {
				shift = queryLen
			}
			pos += max(1, shift)
		}
	}

	return matches
}

func (m *metricsModel) highlightMatches(content string) string {
	if m.searchQuery == "" || len(m.searchMatches) == 0 {
		return content
	}

	startLine := m.viewport.YOffset
	endLine := min(len(m.lineOffsets)-1, startLine+m.viewport.Height)

	var result strings.Builder
	result.Grow((endLine - startLine + 1) * 200)

	if startLine > 0 {
		result.WriteString(strings.Repeat("\n", startLine))
	}

	for i := startLine; i <= endLine; i++ {
		lineStart := m.lineOffsets[i]
		lineEnd := len(content)
		if i+1 < len(m.lineOffsets) {
			lineEnd = m.lineOffsets[i+1] - 1
		}

		lineText := content[lineStart:lineEnd]
		truncationOffset := 0

		maxLineLen := m.viewport.Width * 10
		if len(lineText) > maxLineLen {
			xOffset := m.viewport.xOffset
			visibleStart := max(0, xOffset-100)
			visibleEnd := min(len(lineText), xOffset+maxLineLen)

			if visibleEnd <= visibleStart {
				visibleEnd = min(len(lineText), visibleStart+maxLineLen)
			}

			if visibleStart >= len(lineText) {
				visibleStart = max(0, len(lineText)-maxLineLen)
			}
			if visibleEnd > len(lineText) {
				visibleEnd = len(lineText)
			}
			if visibleStart >= visibleEnd {
				visibleStart = 0
				visibleEnd = min(len(lineText), maxLineLen)
			}

			if visibleStart > 0 {
				result.WriteString(strings.Repeat(" ", visibleStart))
			}

			truncationOffset = visibleStart
			lineText = lineText[visibleStart:visibleEnd]
		}

		result.WriteString(m.highlightLine(lineText, lineStart, truncationOffset))
		result.WriteByte('\n')
	}

	remaining := (len(m.lineOffsets) - 1) - endLine
	if remaining > 0 {
		result.WriteString(strings.Repeat("\n", remaining))
	}

	return result.String()
}

func (m *metricsModel) highlightHexMatches() string {
	startRow := m.hexViewport.YOffset
	endRow := min(m.totalHexRows-1, startRow+m.hexViewport.Height+2)

	var result strings.Builder
	result.Grow((endRow - startRow + 1) * 80)

	if startRow > 0 {
		result.WriteString(strings.Repeat("\n", startRow))
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
	if remaining > 0 {
		result.WriteString(strings.Repeat("\n", remaining))
	}

	return result.String()
}

func (m *metricsModel) highlightLine(lineText string, lineStartOffset int, truncationOffset int) string {
	adjustedLineStart := lineStartOffset + truncationOffset
	lineEndOffset := adjustedLineStart + len(lineText)

	startIdx := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].end > adjustedLineStart
	})

	endIdx := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].start >= lineEndOffset
	})

	if startIdx >= endIdx || startIdx >= len(m.searchMatches) {
		return lineText
	}

	maxMatchesToProcess := 200
	totalMatches := endIdx - startIdx

	if totalMatches > maxMatchesToProcess {
		currentMatchInRange := m.currentMatch >= startIdx && m.currentMatch < endIdx

		if currentMatchInRange {
			windowRadius := maxMatchesToProcess / 2
			newStartIdx := max(startIdx, m.currentMatch-windowRadius)
			newEndIdx := min(endIdx, m.currentMatch+windowRadius)
			startIdx = newStartIdx
			endIdx = newEndIdx
		} else {
			endIdx = min(endIdx, startIdx+maxMatchesToProcess)
		}
	}

	var result strings.Builder
	result.Grow(len(lineText) + (endIdx-startIdx)*20)

	lastPos := 0

	for j := startIdx; j < endIdx; j++ {
		match := m.searchMatches[j]

		matchStart := match.start - adjustedLineStart
		matchEnd := match.end - adjustedLineStart

		if matchStart < 0 {
			matchStart = 0
		}
		if matchEnd > len(lineText) {
			matchEnd = len(lineText)
		}
		if matchStart >= len(lineText) {
			break
		}
		if matchEnd <= matchStart {
			continue
		}

		if matchStart > lastPos {
			result.WriteString(valueStyle.Width(0).Render(lineText[lastPos:matchStart]))
		}

		matchedText := lineText[matchStart:matchEnd]
		if j == m.currentMatch {
			result.WriteString(currentMatchStyle.Render(matchedText))
		} else {
			result.WriteString(searchHighlightStyle.Render(matchedText))
		}

		lastPos = matchEnd
	}

	if lastPos < len(lineText) {
		result.WriteString(valueStyle.Width(0).Render(lineText[lastPos:]))
	}

	return result.String()
}

func (m *metricsModel) highlightHexLine(lineText string, lineStartByte int) string {
	lineEndByte := lineStartByte + hexColumnSize

	startIdx := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].end > lineStartByte
	})

	endIdx := sort.Search(len(m.searchMatches), func(i int) bool {
		return m.searchMatches[i].start >= lineEndByte
	})

	if startIdx >= endIdx || startIdx >= len(m.searchMatches) {
		return lineText
	}

	maxMatchesToProcess := 50
	totalMatches := endIdx - startIdx

	if totalMatches > maxMatchesToProcess {
		currentMatchInRange := m.currentMatch >= startIdx && m.currentMatch < endIdx

		if currentMatchInRange {
			windowRadius := maxMatchesToProcess / 2
			newStartIdx := max(startIdx, m.currentMatch-windowRadius)
			newEndIdx := min(endIdx, m.currentMatch+windowRadius)
			startIdx = newStartIdx
			endIdx = newEndIdx
		} else {
			endIdx = min(endIdx, startIdx+maxMatchesToProcess)
		}
	}

	byteCount := (len(lineText) + 1) / 3

	var result strings.Builder
	result.Grow(len(lineText) + (endIdx-startIdx)*10)

	lastByte := 0

	for j := startIdx; j < endIdx; j++ {
		match := m.searchMatches[j]

		matchStartByte := match.start - lineStartByte
		matchEndByte := match.end - lineStartByte

		if matchStartByte < 0 {
			matchStartByte = 0
		}
		if matchEndByte > byteCount {
			matchEndByte = byteCount
		}
		if matchStartByte >= byteCount {
			break
		}
		if matchEndByte <= matchStartByte {
			continue
		}

		for i := lastByte; i < matchStartByte; i++ {
			charIdx := i * 3
			if charIdx+2 <= len(lineText) {
				result.WriteString(lineText[charIdx : charIdx+2])
				if charIdx+2 < len(lineText) {
					result.WriteByte(' ')
				}
			}
		}

		var matchedHex strings.Builder
		for i := matchStartByte; i < matchEndByte; i++ {
			charIdx := i * 3
			if charIdx+2 <= len(lineText) {
				matchedHex.WriteString(lineText[charIdx : charIdx+2])
				if i < matchEndByte-1 {
					matchedHex.WriteByte(' ')
				}
			}
		}

		if j == m.currentMatch {
			result.WriteString(currentMatchStyle.Render(matchedHex.String()))
		} else {
			result.WriteString(searchHighlightStyle.Render(matchedHex.String()))
		}

		if matchEndByte < byteCount {
			result.WriteByte(' ')
		}

		lastByte = matchEndByte
	}

	var unmatchedHex strings.Builder
	for i := lastByte; i < byteCount; i++ {
		charIdx := i * 3
		if charIdx+2 <= len(lineText) {
			unmatchedHex.WriteString(lineText[charIdx : charIdx+2])
			if charIdx+2 < len(lineText) {
				unmatchedHex.WriteByte(' ')
			}
		}
	}
	result.WriteString(valueStyle.Width(0).Render(unmatchedHex.String()))

	return result.String()
}

func (m *metricsModel) jumpToMatch() {
	if len(m.searchMatches) == 0 || m.currentMatch >= len(m.searchMatches) {
		return
	}
	match := m.searchMatches[m.currentMatch]

	lineStart := m.lineOffsets[match.line]
	matchColumn := match.start - lineStart

	hexRow := match.start / hexColumnSize
	hexCol := match.start % hexColumnSize

	targetY := max(0, match.line-(m.viewport.Height/2))
	m.viewport.SetYOffset(targetY)

	targetX := max(0, matchColumn-(m.viewport.Width/2))
	m.viewport.xOffset = targetX

	targetHexY := max(0, hexRow-(m.hexViewport.Height/2))
	m.hexViewport.YOffset = targetHexY

	hexCharPos := hexCol * 3
	targetHexX := max(0, hexCharPos-(m.hexViewport.Width/2))
	m.hexViewport.xOffset = targetHexX

	m.refreshViewportContent()
}

func getLineOffsets(content string) []int {
	estimatedLines := len(content)/80 + 100
	offsets := make([]int, 0, estimatedLines)
	offsets = append(offsets, 0)

	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			offsets = append(offsets, i+1)
		}
	}

	return offsets
}
