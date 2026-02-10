package search

import (
	"sort"
	"strings"
)

type Match struct {
	Line  int
	Start int
	End   int
}

func FindMatches(content, query string, lineOffsets []int, normalCase bool) []Match {
	if query == "" || len(lineOffsets) == 0 {
		return nil
	}

	if !normalCase {
		content = strings.ToLower(content)
		query = strings.ToLower(strings.ReplaceAll(query, "\\n", "\n"))
	}

	queryLen := len(query)
	contentLen := len(content)

	if queryLen == 0 || queryLen > contentLen {
		return nil
	}

	matches := make([]Match, 0, min(1000, contentLen/100))

	badChar := make(map[byte]int)
	for i := range queryLen {
		if i < queryLen-1 {
			badChar[query[i]] = queryLen - 1 - i
		}
	}

	findLine := func(pos int) int {
		return sort.Search(len(lineOffsets), func(i int) bool {
			return lineOffsets[i] > pos
		}) - 1
	}

	pos := 0
	for pos <= contentLen-queryLen {
		j := queryLen - 1
		for j >= 0 && content[pos+j] == query[j] {
			j--
		}

		if j < 0 {
			actualStart := pos
			actualEnd := pos + queryLen
			lineIdx := findLine(actualStart)

			matches = append(matches, Match{
				Line:  lineIdx,
				Start: actualStart,
				End:   actualEnd,
			})
			pos += queryLen
		} else {
			shift, ok := badChar[content[pos+queryLen-1]]
			if !ok {
				shift = queryLen
			}
			pos += max(1, shift)
		}
	}

	return matches
}

func GetLineOffsets(content string) ([]int, int) {
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
