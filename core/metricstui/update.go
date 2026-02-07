package metricstui

import (
	"fmt"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/stream"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd = m.handleWindowSize(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		cmd = m.handleKeyPress(msg)
		if cmd != nil {
			return m, cmd
		}

	case MetricsMsg:
		m.handleMetrics(msg)

	case *stream.Response:
		return m, m.handleHTTPResponse(msg)

	case httpLogReadyMsg:
		m.handleHTTPLogReady(msg)

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *metricsModel) handleWindowSize(msg tea.WindowSizeMsg) tea.Cmd {
	m.viewWidth = 50
	m.viewHeight = 20
	return nil
}

func (m *metricsModel) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	if m.searchMode {
		return m.handleDetailViewKeys(msg)
	}

	if key.Matches(msg, m.keys.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return nil
	}

	if key.Matches(msg, m.keys.Quit) {
		return tea.Quit
	}

	if key.Matches(msg, m.keys.Back) {
		if m.searchMode {
		} else if m.viewMode == ViewModeDetail {
			m.viewMode = ViewModeList
			return nil
		}
	}

	if m.viewMode == ViewModeDetail && key.Matches(msg, m.keys.Tab) {
		m.activeTab = (m.activeTab + 1) % 2
		m.setSelectedLog()
		return nil
	}

	switch m.viewMode {
	case ViewModeList:
		return m.handleListViewKeys(msg)
	case ViewModeDetail:
		return m.handleDetailViewKeys(msg)
	}

	return nil
}

func (m *metricsModel) handleListViewKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.logStore.MoveUp()
	case key.Matches(msg, m.keys.Down):
		m.logStore.MoveDown()
	case key.Matches(msg, m.keys.Enter):
		if m.logStore.Len() > 0 {
			m.setSelectedLog()
			m.currentMatch = 0
			m.viewMode = ViewModeDetail
			m.activeTab = TabResponseBody
		}
	}
	return nil
}

func (m *metricsModel) handleDetailViewKeys(msg tea.KeyMsg) tea.Cmd {
	if m.searchMode {
		return m.handleSearchInput(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Search):
		m.searchMode = true
		return nil

	case key.Matches(msg, m.keys.NextMatch):
		if len(m.searchMatches) > 0 {
			m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
			m.jumpToMatch()
		}
		return nil

	case key.Matches(msg, m.keys.PrevMatch):
		if len(m.searchMatches) > 0 {
			m.currentMatch = (m.currentMatch - 1 + len(m.searchMatches)) % len(m.searchMatches)
			m.jumpToMatch()
		}
		return nil

	case key.Matches(msg, m.keys.Up):
		m.textYOffset = max(0, m.textYOffset-1)
		m.hexYOffset = max(0, m.hexYOffset-1)
		m.xOffset = 0
		return nil

	case key.Matches(msg, m.keys.Down):
		maxTextY := max(0, len(m.lineOffsets)-m.viewHeight)
		maxHexY := max(0, m.totalHexRows-m.viewHeight)
		m.textYOffset = min(maxTextY, m.textYOffset+1)
		m.hexYOffset = min(maxHexY, m.hexYOffset+1)
		return nil

	case key.Matches(msg, m.keys.GotoTop):
		if time.Since(m.lastGPress) < 500*time.Millisecond {
			m.textYOffset = 0
			m.hexYOffset = 0
		}
		m.lastGPress = time.Now()
		return nil

	case key.Matches(msg, m.keys.GotoBottom):
		m.textYOffset = max(0, len(m.lineOffsets)-m.viewHeight)
		m.hexYOffset = max(0, m.totalHexRows-m.viewHeight)
		return nil

	case key.Matches(msg, m.keys.GoToLeft):
		m.xOffset = 0
		return nil

	case key.Matches(msg, m.keys.GoToRight):
		maxScroll := max(0, m.maxLineLength-m.viewWidth)
		m.xOffset = maxScroll
		return nil

	case key.Matches(msg, m.keys.Left):
		m.xOffset = max(0, m.xOffset-1)
		return nil

	case key.Matches(msg, m.keys.Right):
		maxHorizScroll := max(0, m.maxLineLength-m.viewWidth)
		m.xOffset = min(maxHorizScroll, m.xOffset+1)
		return nil
	}

	return nil
}

func (m *metricsModel) handleSearchInput(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.CancelSearch):
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.currentMatch = 0
			m.searchMatches = nil
		} else {
			m.searchMode = false
		}
		return nil

	case key.Matches(msg, m.keys.Enter):
		if len(m.searchMatches) > 0 {
			m.currentMatch = 0
			m.jumpToMatch()
		}
		m.searchMode = false
		return nil

	case msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.findMatches()
		}
		return nil

	case msg.Type == tea.KeyRunes || msg.String() == " ":
		m.searchQuery += msg.String()
		m.findMatches()
		return nil
	}

	return nil
}

func (m *metricsModel) handleMetrics(msg MetricsMsg) {
	m.metricsData.UpdateMetrics(msg)
}

func (m *metricsModel) handleHTTPResponse(msg *stream.Response) tea.Cmd {
	log := &HTTPLogMsg{response: msg}

	responseBody, err := readResponseBody(msg)
	if err != nil {
		responseBody = fmt.Appendf(nil, "Error reading body: %v", err)
	}
	log.responseBody = responseBody

	return func() tea.Msg {
		var protoLog *proto.HTTPLog

		select {
		case request := <-m.requestch:
			log.request = request
			requestBody, err := readRequestBody(request)
			if err != nil {
				requestBody = fmt.Appendf(nil, "Error reading body: %v", err)
			}
			log.requestBody = requestBody
		case <-time.After(5 * time.Second):
		}

		select {
		case protoLog = <-m.httpLogch:
		case <-time.After(5 * time.Second):
			return nil
		}

		log.HTTPLog = protoLog

		return httpLogReadyMsg{log: log}
	}
}

func (m *metricsModel) handleHTTPLogReady(msg httpLogReadyMsg) {
	if msg.log == nil {
		return
	}

	m.logStore.AddLog(msg.log)
}
