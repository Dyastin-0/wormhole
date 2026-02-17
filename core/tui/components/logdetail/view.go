package logdetail

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/formatters"
	"github.com/Dyastin-0/wormhole/core/tui/search"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.log == nil {
		return styles.Title.Render("No request selected")
	}

	headerCol := m.renderHeaderColumn()
	bodyCol := m.renderBodyColumn()

	return lipgloss.JoinHorizontal(lipgloss.Top, headerCol, "  ", bodyCol)
}

func (m Model) renderMetadata(title string) string {
	statusValue := formatters.FormatStatusCode(m.log.Status, false)

	metaLines := []string{
		styles.Title.Foreground(styles.Highlight).Render(title),
		"",
		m.formatDetailLine(
			"Timestamp",
			styles.LogTime.Width(styles.TimeWidth+11).Render(
				time.Unix(m.log.Timestamp, 0).Format("2006-01-02 15:04:05"),
			),
		),
		m.formatDetailLine(
			"Method",
			styles.LogMethod.Render(m.log.Method),
		),
		m.formatDetailLine(
			"Path",
			styles.LogPath.Render(m.log.Path),
		),
		m.formatDetailLine(
			"Status",
			statusValue,
		),
		m.formatDetailLine(
			"Size",
			// RespBody length is the closest equivalent to the old response.Size
			styles.LogSize.Render(formatters.FormatBytes(uint64(len(m.log.RespBody)))),
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

// loadContent pulls body and header content from proto.HTTPLog directly.
func (m *Model) loadContent() {
	if m.log == nil {
		return
	}

	if m.activeTab == tabRequestBody {
		m.stringContent = string(m.log.ReqBody)
		m.headerContent = MergeHeaders(m.log.ReqHeaders)
	} else {
		m.stringContent = string(m.log.RespBody)
		m.headerContent = MergeHeaders(m.log.RespHeaders)
	}

	m.headerLineOffsets, m.maxHeaderLineLength = search.GetLineOffsets(m.headerContent)
	m.lineOffsets, m.maxLineLength = search.GetLineOffsets(m.stringContent)
	m.totalHexRows = (len(m.stringContent) + styles.HexColumnSize - 1) / styles.HexColumnSize

	if m.wrapBody {
		m.visualLines = search.GetWrappedLines(m.stringContent, m.lineOffsets, m.viewWidth)
		m.headerVisualLines = search.GetWrappedLines(m.headerContent, m.headerLineOffsets, m.headerViewWidth)
	} else {
		m.visualLines = nil
		m.headerVisualLines = nil
	}

	switch m.focusedPanel {
	case focusHeaderPanel:
		m.headerYOffset = 0
		m.headerXOffset = 0
	case focusBodyPanel:
		m.textYOffset = 0
		m.hexYOffset = 0
		m.xOffset = 0
	}

	m.findMatches()
}

func (m Model) renderBodyColumn() string {
	// Content-Type comes from RespHeaders or ReqHeaders depending on active tab.
	var contentType string
	if m.activeTab == tabRequestBody {
		contentType = m.log.ReqHeaders.Get("Content-Type")
	} else {
		contentType = m.log.RespHeaders.Get("Content-Type")
	}

	isText := contentType == "" ||
		strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "application/javascript") ||
		strings.HasPrefix(contentType, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(contentType, "application/graphql") ||
		strings.Contains(contentType, "+json") ||
		strings.Contains(contentType, "+xml")

	var matches []search.Match
	if m.focusedPanel == focusBodyPanel {
		matches = m.bodySearchMatches
	}

	var textView, hexView string

	if m.displayText && isText {
		if m.wrapBody {
			textView = search.HighlightWrappedMatches(
				m.stringContent, m.visualLines, matches,
				m.bodyCurrentMatch, m.textYOffset, m.viewHeight,
				max(minTextViewWidth, m.viewWidth),
			)
		} else {
			textView = search.HighlightMatches(
				m.stringContent, m.lineOffsets, matches,
				m.bodyCurrentMatch, m.textYOffset, m.viewHeight,
				max(minTextViewWidth, m.viewWidth), m.xOffset,
			)
		}
	}

	if m.displayHex {
		hexView = search.HighlightHexMatches(
			m.stringContent, matches, m.bodyCurrentMatch,
			m.hexYOffset, m.viewHeight, styles.HexColumnSize,
		)
	}

	showText := m.displayText && isText

	var viewports string
	if showText && m.displayHex {
		text := styles.Text.Width(m.viewWidth).MaxWidth(m.viewWidth).Render(textView)
		hex := styles.Text.Render(hexView)
		viewports = lipgloss.JoinHorizontal(lipgloss.Top, text, "  ", hex)
	} else if showText {
		viewports = styles.Text.Width(m.viewWidth).MaxWidth(m.viewWidth).Render(textView)
	} else if m.displayHex {
		viewports = styles.Text.Render(hexView)
	}

	bodyIndicator := " "
	if m.focusedPanel == focusBodyPanel {
		bodyIndicator = "█"
	}

	bodyTitleStr := fmt.Sprintf("%s %s ",
		styles.Title.Foreground(styles.Highlight).Render("Body"),
		styles.Title.Render(formatters.FormatBytes(uint64(m.getBodySize()))),
	)
	bodyTitle := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.Title.Render(bodyTitleStr),
		styles.HelpKey.Render(bodyIndicator),
	)

	footerRow := m.renderBodyFooter(showText)

	return lipgloss.JoinVertical(lipgloss.Left,
		bodyTitle,
		"",
		viewports,
		footerRow,
	)
}

func (m Model) getBodySize() int {
	if m.log == nil {
		return 0
	}
	if m.activeTab == tabResponseBody {
		return len(m.log.RespBody)
	}
	return len(m.log.ReqBody)
}

// MergeHeaders converts http.Header (map[string][]string) into a sorted,
// human-readable string for display in the header panel.
func MergeHeaders(headers map[string][]string) string {
	if len(headers) == 0 {
		return ""
	}

	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		for _, v := range headers[k] {
			fmt.Fprintf(&sb, "%s: %s\n", k, v)
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// --- unchanged rendering helpers below, kept for completeness ---

func (m Model) renderHeaderColumn() string {
	var title string
	switch m.activeTab {
	case tabResponseBody:
		title = "Response Details"
	case tabRequestBody:
		title = "Request Details"
	}

	meta := m.renderMetadata(title)

	var matches []search.Match
	if m.focusedPanel == focusHeaderPanel {
		matches = m.headerSearchMatches
	}

	var headerView string
	if m.wrapHeaders {
		headerView = search.HighlightWrappedMatches(
			m.headerContent, m.headerVisualLines, matches,
			m.headerCurrentMatch, m.headerYOffset, m.headerViewHeight, m.headerViewWidth,
		)
	} else {
		headerView = search.HighlightMatches(
			m.headerContent, m.headerLineOffsets, matches,
			m.headerCurrentMatch, m.headerYOffset, m.headerViewHeight, m.headerViewWidth, m.headerXOffset,
		)
	}

	headerIndicator := " "
	if m.focusedPanel == focusHeaderPanel {
		headerIndicator = "█"
	}

	headerTitle := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.Title.Foreground(styles.Highlight).Render("Header "),
		styles.HelpKey.Render(headerIndicator),
	)

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		headerTitle,
		"",
		styles.Text.Render(headerView),
	)

	totalRows := len(m.headerLineOffsets)
	if m.wrapHeaders {
		totalRows = len(m.headerVisualLines)
	}

	end := min(totalRows, m.headerYOffset+m.headerViewHeight)
	visible := m.headerYOffset + 1
	if m.maxHeaderLineLength <= 0 {
		visible = 0
	}

	headerScrollInfo := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.HelpKey.Render("Rows"),
		styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", visible, end, totalRows)),
	)

	if !m.wrapHeaders {
		end = min(m.maxHeaderLineLength, m.headerXOffset+m.headerViewWidth)
		visible = m.headerXOffset + 1
		if len(m.lineOffsets) <= 0 {
			visible = 0
		}
		headerScrollInfo = lipgloss.JoinHorizontal(lipgloss.Left,
			headerScrollInfo,
			m.help.Styles.ShortSeparator.Render(m.help.ShortSeparator),
			styles.HelpKey.Render("Cols"),
			styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", visible, end, m.maxHeaderLineLength)),
		)
	}

	footer := m.renderHeaderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		meta,
		header,
		"",
		headerScrollInfo,
		"",
		footer,
	)
}

func (m Model) renderHeaderFooter() string {
	helpView := m.help.View(m.keys)

	var statusLine string
	matches := m.activeSearchMatches()

	if m.searchActive {
		matchInfo := ""
		if len(*matches) > 0 {
			matchInfo = fmt.Sprintf(" (%d/%d)", m.currentMatch()+1, len(*matches))
		}
		searchStr := styles.HelpKey.Render("Search")
		inputStr := m.activeSearchInput().View()
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, searchStr, " ", inputStr, styles.Footer.Render(matchInfo))
	} else if len(*matches) > 0 {
		matchStr := styles.HelpKey.Render("Match")
		statusLine = styles.Value.Width(0).Render(fmt.Sprintf(" %d/%d", m.currentMatch()+1, len(*matches)))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, matchStr, statusLine)
	}

	if statusLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left, statusLine, helpView)
	}

	return helpView
}

func (m Model) renderBodyFooter(showText bool) string {
	var scrollInfoParts []string

	if showText {
		totalRows := len(m.lineOffsets)
		if m.wrapBody {
			totalRows = len(m.visualLines)
		}
		currentRowEnd := min(totalRows, m.textYOffset+m.viewHeight)
		visible := m.textYOffset + 1
		if len(m.lineOffsets) <= 0 {
			visible = 0
		}
		textYInfo := lipgloss.JoinHorizontal(lipgloss.Left,
			styles.HelpKey.Render("Text rows"),
			styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", visible, currentRowEnd, totalRows)),
		)

		var textXInfo string
		if !m.wrapBody {
			rightCol := min(m.maxLineLength, m.xOffset+m.viewWidth)
			visible = m.xOffset + 1
			if m.maxLineLength <= 0 {
				visible = 0
			}
			textXInfo = lipgloss.JoinHorizontal(lipgloss.Left,
				styles.HelpKey.Render("Text cols"),
				styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", visible, rightCol, m.maxLineLength)),
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
		visible := m.hexYOffset + 1
		if m.totalHexRows <= 0 {
			visible = 0
		}
		hexYInfo := lipgloss.JoinHorizontal(lipgloss.Left,
			styles.HelpKey.Render("Hex rows"),
			styles.Footer.Render(fmt.Sprintf(" %d-%d of %d", visible, bottomHex, m.totalHexRows)),
		)
		scrollInfoParts = append(scrollInfoParts, hexYInfo)
	}

	if len(scrollInfoParts) == 0 {
		return ""
	}

	return lipgloss.NewStyle().MarginTop(1).Render(
		lipgloss.JoinVertical(lipgloss.Left, scrollInfoParts...),
	)
}

func (m Model) formatDetailLine(label, styledValue string) string {
	l := styles.DetailLabel.Render(label)
	leftAlignedLabel := lipgloss.PlaceHorizontal(styles.LabelWidth-1, lipgloss.Left, l)
	fullLabel := fmt.Sprintf("%s:", leftAlignedLabel)
	return lipgloss.JoinHorizontal(lipgloss.Left, fullLabel, " ", styledValue)
}

func (m Model) currentMatch() int {
	switch m.focusedPanel {
	case focusHeaderPanel:
		return m.headerCurrentMatch
	case focusBodyPanel:
		return m.bodyCurrentMatch
	}
	return 0
}
