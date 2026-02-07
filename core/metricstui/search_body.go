package metricstui

import (
	"sort"
	"strings"
)

func (m *metricsModel) findMatches() {
	if m.searchQuery == "" {
		m.searchMatches = nil
		m.currentMatch = 0
		return
	}

	query := strings.ToLower(strings.ReplaceAll(m.searchQuery, "\\n", "\n"))
	queryLen := len(query)

	matches := m.findMatchesBMH(query, queryLen)
	m.searchMatches = matches
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
	startLine := m.textYOffset
	var result strings.Builder
	result.Grow(m.viewHeight * (m.viewWidth + 1))

	for i := 0; i < m.viewHeight; i++ {
		lineIdx := startLine + i

		if lineIdx >= len(m.lineOffsets) {
			result.WriteString(strings.Repeat(" ", m.viewWidth))
			if i < m.viewHeight-1 {
				result.WriteByte('\n')
			}
			continue
		}

		lineStart := m.lineOffsets[lineIdx]
		lineEnd := len(content)
		if lineIdx+1 < len(m.lineOffsets) {
			lineEnd = m.lineOffsets[lineIdx+1] - 1
		}

		lineText := content[lineStart:lineEnd]
		visibleStart := min(len(lineText), m.xOffset)
		visibleEnd := min(len(lineText), m.xOffset+m.viewWidth)

		displayedLen := 0
		if visibleStart < visibleEnd {
			segment := lineText[visibleStart:visibleEnd]
			displayedLen = len(segment)
			if m.searchQuery != "" && len(m.searchMatches) > 0 {
				result.WriteString(m.highlightLine(segment, lineStart, visibleStart))
			} else {
				result.WriteString(segment)
			}
		}

		if displayedLen < m.viewWidth {
			result.WriteString(strings.Repeat(" ", m.viewWidth-displayedLen))
		}

		if i < m.viewHeight-1 {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func (m *metricsModel) highlightHexMatches() string {
	startRow := m.hexYOffset
	var result strings.Builder

	for i := 0; i < m.viewHeight; i++ {
		currentRow := startRow + i
		rowStartByte := currentRow * hexColumnSize

		if rowStartByte >= len(m.stringContent) {
			result.WriteString(strings.Repeat(" ", (hexColumnSize*3)-1))
		} else {
			rowEndByte := min(rowStartByte+hexColumnSize, len(m.stringContent))
			lineText := hexLine(m.stringContent[rowStartByte:rowEndByte])

			if m.searchQuery != "" && len(m.searchMatches) > 0 {
				result.WriteString(m.highlightHexLine(lineText, rowStartByte))
			} else {
				result.WriteString(lineText)
			}
		}

		if i < m.viewHeight-1 {
			result.WriteByte('\n')
		}
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

	maxMatchesToProcess := 300
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

	maxTextY := max(0, len(m.lineOffsets)-m.viewHeight)
	maxHexY := max(0, m.totalHexRows-m.viewHeight)

	targetY := match.line - (m.viewHeight / 2)
	hexRow := match.start / hexColumnSize
	targetHexY := hexRow - (m.viewHeight / 2)

	m.textYOffset = max(0, min(targetY, maxTextY))
	m.hexYOffset = max(0, min(targetHexY, maxHexY))

	lineStart := m.lineOffsets[match.line]
	matchX := match.start - lineStart

	targetX := matchX - (m.viewWidth / 2)

	maxTextX := max(0, m.maxLineLength-m.viewWidth)
	m.xOffset = max(0, min(targetX, maxTextX))
}

func getLineOffsets(content string) ([]int, int) {
	estimatedLines := len(content)/80 + 100
	offsets := make([]int, 0, estimatedLines)
	offsets = append(offsets, 0)

	maxLineLength := 0
	lastOffset := 0

	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lineLength := i - lastOffset
			if lineLength > maxLineLength {
				maxLineLength = lineLength
			}
			offsets = append(offsets, i+1)
			lastOffset = i + 1
		}
	}

	if lastOffset < len(content) {
		lineLength := len(content) - lastOffset
		if lineLength > maxLineLength {
			maxLineLength = lineLength
		}
	}

	return offsets, maxLineLength
}
