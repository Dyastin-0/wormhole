package logdetail

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/formatters"
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/search"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.log == nil {
		return styles.Title.Render("No request selected")
	}

	var title string
	var headerLines string

	switch m.activeTab {
	case messages.TabResponseBody:
		title = "Response Details"
		headerLines = formatters.SortAndRenderHeaders(m.log.Response.Header)
	case messages.TabRequestBody:
		title = "Request Details"
		headerLines = formatters.SortAndRenderHeaders(m.log.Request.Header)
	}

	meta := m.renderMetadata(title)
	footer := m.renderFooter()

	headerColumn := lipgloss.JoinVertical(lipgloss.Left,
		meta,
		headerLines,
		footer,
	)

	bodyColumn := m.renderBodyColumn()

	return lipgloss.JoinHorizontal(lipgloss.Left, headerColumn, "  ", bodyColumn)
}

func (m Model) renderMetadata(title string) string {
	statusValue := formatters.FormatStatusCode(m.log.Response.StatusCode, false)

	metaLines := []string{
		styles.Title.Render(title),
		"",
		m.formatDetailLine(
			"Timestamp",
			styles.LogTime.Width(styles.TimeWidth+11).Render(
				time.Unix(m.log.Timestamp, 0).Format("2006-01-02 15:04:05"),
			),
		),
		m.formatDetailLine(
			"Method",
			styles.LogMethod.Render(m.log.Request.Method),
		),
		m.formatDetailLine(
			"Path",
			styles.LogPath.Render(m.log.Request.URL.Path),
		),
		m.formatDetailLine(
			"Status",
			statusValue,
		),
		m.formatDetailLine(
			"Size",
			styles.LogSize.Render(formatters.FormatBytes(uint64(m.log.Response.Size))),
		),
		m.formatDetailLine(
			"Duration",
			styles.LogDuration.Align(lipgloss.Left).Render(
				fmt.Sprintf("%.2f ms", float64(m.log.Duration)/1000.0),
			),
		),
		"",
	}

	return lipgloss.JoinVertical(lipgloss.Left, metaLines...)
}

func (m Model) renderFooter() string {
	m.keys.Search.SetEnabled(!m.searchActive)
	helpView := m.help.View(m.keys)

	var statusLine string
	if m.searchActive {
		matchInfo := ""
		if len(m.searchMatches) > 0 {
			matchInfo = fmt.Sprintf(" (%d/%d)", m.currentMatch+1, len(m.searchMatches))
		}
		searchStr := styles.HelpKey.Render("Search")
		inputStr := styles.Value.Width(0).Render(fmt.Sprintf(" /%s%s", m.searchInput.Value(), matchInfo))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, searchStr, inputStr)
	} else if len(m.searchMatches) > 0 {
		matchStr := styles.HelpKey.Render("Match")
		statusLine = styles.Value.Width(0).Render(fmt.Sprintf(" %d/%d", m.currentMatch+1, len(m.searchMatches)))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, matchStr, statusLine)
	}

	if statusLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			statusLine,
			helpView,
		)
	}

	return helpView
}

func (m Model) renderBodyColumn() string {
	var textView, hexView string
	contentType := m.log.Response.Header.Get("Content-Type")
	isText := contentType == "" ||
		strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "application/javascript") ||
		strings.HasPrefix(contentType, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(contentType, "application/graphql") ||
		strings.Contains(contentType, "+json") ||
		strings.Contains(contentType, "+xml")

	if m.displayText && isText {
		if m.wrapText {
			textView = search.HighlightWrappedMatches(
				m.stringContent,
				m.visualLines,
				m.searchMatches,
				m.currentMatch,
				m.textYOffset,
				m.viewHeight,
				m.viewWidth,
			)
		} else {
			textView = search.HighlightMatches(
				m.stringContent,
				m.lineOffsets,
				m.searchMatches,
				m.currentMatch,
				m.textYOffset,
				m.viewHeight,
				m.viewWidth,
				m.xOffset,
			)
		}
	}

	if m.displayHex {
		hexView = search.HighlightHexMatches(
			m.stringContent,
			m.searchMatches,
			m.currentMatch,
			m.hexYOffset,
			m.viewHeight,
			styles.HexColumnSize,
		)
	}

	var viewports string
	showText := m.displayText && isText

	if showText && m.displayHex {
		textTitle := styles.Value.Width(0).Render("ascii")
		textWithTitle := lipgloss.JoinVertical(lipgloss.Left, textTitle, "", textView)
		text := styles.Text.Width(m.viewWidth).MaxWidth(m.viewWidth).Render(textWithTitle)

		hexTitle := styles.Value.Width(0).Render("hex")
		hexWithTitle := lipgloss.JoinVertical(lipgloss.Left, hexTitle, "", hexView)
		hex := styles.Text.Render(hexWithTitle)

		viewports = lipgloss.JoinHorizontal(lipgloss.Top, text, "  ", hex)
	} else if showText {
		textTitle := styles.Value.Width(0).Render("ascii")
		textWithTitle := lipgloss.JoinVertical(lipgloss.Left, textTitle, "", textView)
		viewports = styles.Text.Width(m.viewWidth).MaxWidth(m.viewWidth).Render(textWithTitle)
	} else if m.displayHex {
		hexTitle := styles.Value.Width(0).Render("hex")
		hexWithTitle := lipgloss.JoinVertical(lipgloss.Left, hexTitle, "", hexView)
		viewports = styles.Text.Render(hexWithTitle)
	}

	var scrollInfoParts []string

	if showText {
		totalRows := len(m.lineOffsets)
		if m.wrapText {
			totalRows = len(m.visualLines)
		}
		currentRowEnd := min(totalRows, m.textYOffset+m.viewHeight)
		textYInfo := lipgloss.JoinHorizontal(lipgloss.Left,
			styles.HelpKey.Render("Text rows"),
			styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", m.textYOffset+1, currentRowEnd, totalRows)),
		)

		var textXInfo string
		if !m.wrapText {
			rightCol := min(m.maxLineLength, m.xOffset+m.viewWidth)
			textXInfo = lipgloss.JoinHorizontal(lipgloss.Left,
				styles.HelpKey.Render("Text cols"),
				styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", m.xOffset+1, rightCol, m.maxLineLength)),
			)
		}

		if textXInfo != "" {
			sep := m.help.Styles.ShortSeparator.Render(m.help.ShortSeparator)
			scrollInfoParts = append(scrollInfoParts, lipgloss.JoinHorizontal(lipgloss.Left, textYInfo, sep, textXInfo))
		} else {
			scrollInfoParts = append(scrollInfoParts, textYInfo)
		}
	}

	if m.displayHex {
		bottomHex := min(m.totalHexRows, m.hexYOffset+m.viewHeight)
		hexYInfo := lipgloss.JoinHorizontal(lipgloss.Left,
			styles.HelpKey.Render("Hex rows"),
			styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", m.hexYOffset+1, bottomHex, m.totalHexRows)),
		)
		scrollInfoParts = append(scrollInfoParts, hexYInfo)
	}

	var footerRow string
	if len(scrollInfoParts) > 0 {
		scrollInfo := lipgloss.JoinVertical(lipgloss.Left, scrollInfoParts...)
		footerRow = lipgloss.NewStyle().MarginTop(1).Render(scrollInfo)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		styles.Value.Width(0).Render(fmt.Sprintf("Body %s", formatters.FormatBytes(uint64(m.getBodySize())))),
		"",
		viewports,
		footerRow,
	)
}

func (m Model) formatDetailLine(label, styledValue string) string {
	l := styles.DetailLabel.Render(label)
	leftAlignedLabel := lipgloss.PlaceHorizontal(styles.LabelWidth-1, lipgloss.Left, l)
	fullLabel := fmt.Sprintf("%s:", leftAlignedLabel)
	return lipgloss.JoinHorizontal(lipgloss.Left, fullLabel, " ", styledValue)
}

func (m *Model) loadContent() {
	if m.log == nil {
		return
	}

	var content string
	if m.activeTab == messages.TabRequestBody {
		content = string(m.log.RequestBody)
	} else {
		content = string(m.log.ResponseBody)
	}

	m.stringContent = content
	m.lineOffsets, m.maxLineLength = search.GetLineOffsets(m.stringContent)
	m.totalHexRows = (len(m.stringContent) + styles.HexColumnSize - 1) / styles.HexColumnSize

	if m.wrapText {
		m.visualLines = search.GetWrappedLines(m.stringContent, m.lineOffsets, m.viewWidth)
	} else {
		m.visualLines = nil
	}

	m.textYOffset = 0
	m.hexYOffset = 0
	m.xOffset = 0

	m.findMatches()
}

func (m Model) getBodySize() int {
	if m.log == nil {
		return 0
	}
	if m.activeTab == messages.TabResponseBody {
		return len(m.log.ResponseBody)
	}
	return len(m.log.RequestBody)
}
