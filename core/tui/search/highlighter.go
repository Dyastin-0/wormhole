package search

import (
	"sort"
	"strings"

	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

const hexChars = "0123456789abcdef"

func HexLine(data string) string {
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

func HighlightMatches(content string, lineOffsets []int, matches []Match, currentMatch int, startLine, viewHeight, viewWidth, xOffset int) string {
	var result strings.Builder
	result.Grow(viewHeight * (viewWidth + 1))

	for i := range viewHeight {
		lineIdx := startLine + i

		if lineIdx >= len(lineOffsets) {
			result.WriteString(strings.Repeat(" ", viewWidth))
			if i < viewHeight-1 {
				result.WriteByte('\n')
			}
			continue
		}

		lineStart := lineOffsets[lineIdx]
		lineEnd := len(content)
		if lineIdx+1 < len(lineOffsets) {
			lineEnd = lineOffsets[lineIdx+1] - 1
		}

		lineText := content[lineStart:lineEnd]
		visibleStart := min(len(lineText), xOffset)
		visibleEnd := min(len(lineText), xOffset+viewWidth)

		displayedLen := 0
		if visibleStart < visibleEnd {
			segment := lineText[visibleStart:visibleEnd]
			displayedLen = len(segment)
			if len(matches) > 0 {
				result.WriteString(highlightLine(sanitize(segment), lineStart, visibleStart, matches, currentMatch))
			} else {
				result.WriteString(sanitize(segment))
			}
		}

		if displayedLen < viewWidth {
			result.WriteString(strings.Repeat(" ", viewWidth-displayedLen))
		}

		if i < viewHeight-1 {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func HighlightWrappedMatches(
	content string,
	visualLines []VisualLine,
	matches []Match,
	currentMatch int,
	startY, viewHeight, viewWidth int,
) string {
	var result strings.Builder
	result.Grow(viewHeight * (viewWidth * 2))

	for i := range viewHeight {
		vIdx := startY + i

		if vIdx >= len(visualLines) {
			result.WriteString(strings.Repeat(" ", viewWidth))
		} else {
			vLine := visualLines[vIdx]
			segment := content[vLine.StartOffset : vLine.StartOffset+vLine.Length]
			segment = sanitize(segment)

			highlighted := highlightLine(segment, vLine.StartOffset, 0, matches, currentMatch)
			result.WriteString(highlighted)

			currentLineWidth := lipgloss.Width(highlighted)
			if currentLineWidth < viewWidth {
				result.WriteString(strings.Repeat(" ", viewWidth-currentLineWidth))
			}
		}

		if i < viewHeight-1 {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func highlightLine(lineText string, lineStartOffset int, truncationOffset int, matches []Match, currentMatch int) string {
	adjustedLineStart := lineStartOffset + truncationOffset
	lineEndOffset := adjustedLineStart + len(lineText)

	startIdx := sort.Search(len(matches), func(i int) bool {
		return matches[i].End > adjustedLineStart
	})

	endIdx := sort.Search(len(matches), func(i int) bool {
		return matches[i].Start >= lineEndOffset
	})

	if startIdx >= endIdx || startIdx >= len(matches) {
		return lineText
	}

	maxMatchesToProcess := 200
	totalMatches := endIdx - startIdx

	if totalMatches > maxMatchesToProcess {
		currentMatchInRange := currentMatch >= startIdx && currentMatch < endIdx

		if currentMatchInRange {
			windowRadius := maxMatchesToProcess / 2
			newStartIdx := max(startIdx, currentMatch-windowRadius)
			newEndIdx := min(endIdx, currentMatch+windowRadius)
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
		match := matches[j]

		matchStart := match.Start - adjustedLineStart
		matchEnd := match.End - adjustedLineStart

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
			result.WriteString(styles.Value.Width(0).Render(lineText[lastPos:matchStart]))
		}

		matchedText := lineText[matchStart:matchEnd]
		if j == currentMatch {
			result.WriteString(styles.CurrentMatch.Render(matchedText))
		} else {
			result.WriteString(styles.SearchHighlight.Render(matchedText))
		}

		lastPos = matchEnd
	}

	if lastPos < len(lineText) {
		result.WriteString(styles.Value.Width(0).Render(lineText[lastPos:]))
	}

	return result.String()
}

func HighlightHexMatches(content string, matches []Match, currentMatch int, startRow, viewHeight, hexColumnSize int) string {
	var result strings.Builder

	for i := range viewHeight {
		currentRow := startRow + i
		rowStartByte := currentRow * hexColumnSize

		if rowStartByte >= len(content) {
			result.WriteString(strings.Repeat(" ", (hexColumnSize*3)-1))
		} else {
			rowEndByte := min(rowStartByte+hexColumnSize, len(content))
			lineText := HexLine(content[rowStartByte:rowEndByte])

			if len(matches) > 0 {
				result.WriteString(highlightHexLine(lineText, rowStartByte, hexColumnSize, matches, currentMatch))
			} else {
				result.WriteString(lineText)
			}
		}

		if i < viewHeight-1 {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func highlightHexLine(lineText string, lineStartByte int, hexColumnSize int, matches []Match, currentMatch int) string {
	lineEndByte := lineStartByte + hexColumnSize

	startIdx := sort.Search(len(matches), func(i int) bool {
		return matches[i].End > lineStartByte
	})

	endIdx := sort.Search(len(matches), func(i int) bool {
		return matches[i].Start >= lineEndByte
	})

	if startIdx >= endIdx || startIdx >= len(matches) {
		return lineText
	}

	maxMatchesToProcess := 50
	totalMatches := endIdx - startIdx

	if totalMatches > maxMatchesToProcess {
		currentMatchInRange := currentMatch >= startIdx && currentMatch < endIdx

		if currentMatchInRange {
			windowRadius := maxMatchesToProcess / 2
			newStartIdx := max(startIdx, currentMatch-windowRadius)
			newEndIdx := min(endIdx, currentMatch+windowRadius)
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
		match := matches[j]

		matchStartByte := match.Start - lineStartByte
		matchEndByte := match.End - lineStartByte

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

		var unmatchedHex strings.Builder
		for i := lastByte; i < matchStartByte; i++ {
			charIdx := i * 3
			if charIdx+2 <= len(lineText) {
				unmatchedHex.WriteString(lineText[charIdx : charIdx+2])
				if charIdx+2 < len(lineText) {
					unmatchedHex.WriteByte(' ')
				}
			}
		}

		result.WriteString(styles.Value.Width(0).Render(unmatchedHex.String()))

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

		if j == currentMatch {
			result.WriteString(styles.CurrentMatch.Render(matchedHex.String()))
		} else {
			result.WriteString(styles.SearchHighlight.Render(matchedHex.String()))
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
	result.WriteString(styles.Value.Width(0).Render(unmatchedHex.String()))

	return result.String()
}

type VisualLine struct {
	StartOffset int
	Length      int
	SourceLine  int
}

func GetWrappedLines(content string, lineOffsets []int, viewWidth int) []VisualLine {
	var visualLines []VisualLine

	for lineIdx, start := range lineOffsets {
		end := len(content)
		if lineIdx+1 < len(lineOffsets) {
			end = lineOffsets[lineIdx+1] - 1
		}

		lineText := content[start:end]
		if len(lineText) == 0 {
			visualLines = append(visualLines, VisualLine{start, 0, lineIdx})
			continue
		}

		for i := 0; i < len(lineText); i += viewWidth {
			size := viewWidth
			if i+size > len(lineText) {
				size = len(lineText) - i
			}
			visualLines = append(visualLines, VisualLine{
				StartOffset: start + i,
				Length:      size,
				SourceLine:  lineIdx,
			})
		}
	}
	return visualLines
}

func JumpToMatch(
	match Match,
	totalLength int,
	lineOffsets []int,
	visualLines []VisualLine,
	wrapText bool,
	viewHeight, viewWidth, hexColumnSize, maxLineLength int,
) (textY, hexY, xOffset int) {
	if wrapText && len(visualLines) > 0 {
		targetRow := 0
		for i, vLine := range visualLines {
			if match.Start >= vLine.StartOffset && match.Start < vLine.StartOffset+vLine.Length {
				targetRow = i
				break
			}
		}
		maxTextY := max(0, len(visualLines)-viewHeight)
		textY = max(0, min(targetRow-(viewHeight/2), maxTextY))
		xOffset = 0
	} else {
		maxTextY := max(0, len(lineOffsets)-viewHeight)
		textY = max(0, min(match.Line-(viewHeight/2), maxTextY))

		lineStart := lineOffsets[match.Line]
		matchX := match.Start - lineStart
		maxTextX := max(0, maxLineLength-viewWidth)
		xOffset = max(0, min(matchX-(viewWidth/2), maxTextX))
	}

	totalHexRows := (totalLength + hexColumnSize - 1) / hexColumnSize
	maxHexY := max(0, totalHexRows-viewHeight)
	hexRow := match.Start / hexColumnSize
	hexY = max(0, min(hexRow-(viewHeight/2), maxHexY))

	return
}

func sanitize(s string) string {
	var result []rune
	for _, r := range s {
		if (r < 32 && r != '\n' && r != '\r') || r == 127 {
			result = append(result, ' ')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
