package metricstui

import "strings"

func (m *metricsModel) highlightMatches(content string) string {
	if m.searchQuery == "" || len(m.searchMatches) == 0 {
		return content
	}

	var result strings.Builder
	result.Grow(len(content) + (len(m.searchMatches) * 20))

	lastIdx := 0

	for i, match := range m.searchMatches {
		start := match.start
		end := match.end

		result.WriteString(content[lastIdx:start])

		target := content[start:end]
		if i == m.currentMatch {
			result.WriteString(currentMatchStyle.Render(target))
		} else {
			result.WriteString(searchHighlightStyle.Render(target))
		}

		lastIdx = end
	}

	result.WriteString(content[lastIdx:])

	return result.String()
}

func (m *metricsModel) findMatches(refresh bool) {
	if m.searchQuery == "" && refresh {
		m.refreshViewportContent()
		return
	}

	query := strings.ToLower(m.searchQuery)
	content := m.lowerCaseContent

	m.searchMatches = make([]*matchLocation, 0, 100)

	offset := 0
	for {
		idx := strings.Index(content[offset:], query)
		if idx == -1 {
			break
		}

		actualIdx := offset + idx

		lineNum := strings.Count(content[:actualIdx], "\n")

		m.searchMatches = append(m.searchMatches, &matchLocation{
			line:  lineNum,
			start: actualIdx,
			end:   actualIdx + len(query),
		})

		offset = actualIdx + len(query)

		if len(m.searchMatches) > 2000 {
			break
		}
	}

	if refresh {
		m.refreshViewportContent()
	}
}

func (m *metricsModel) jumpToMatch() {
	if m.currentMatch >= 0 && m.currentMatch < len(m.searchMatches) {
		match := m.searchMatches[m.currentMatch]
		targetOffset := max(match.line-m.viewport.Height/2, 0)
		m.viewport.SetYOffset(targetOffset)
		m.refreshViewportContent()
	}
}
