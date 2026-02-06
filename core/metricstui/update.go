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
	verticalMarginHeight := headerHeight + footerHeight
	width := min(viewportWidth, msg.Width)

	if !m.ready {
		m.viewport = newViewport(width, msg.Height-verticalMarginHeight)
		m.viewport.Style = valueStyle.Width(0)
		m.viewport.YPosition = headerHeight
		m.hexViewport = newViewport(width, msg.Height-verticalMarginHeight)
		m.hexViewport.Style = valueStyle.Width(0)
		m.hexViewport.YPosition = headerHeight
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = msg.Height - verticalMarginHeight
		m.hexViewport.Width = width
		m.hexViewport.Height = msg.Height - verticalMarginHeight
	}

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

	case key.Matches(msg, m.keys.GotoTop):
		if time.Since(m.lastGPress) < 500*time.Millisecond {
			m.viewport.GotoTop()
			m.hexViewport.GotoTop()
			m.refreshViewportContent()
		}
		m.lastGPress = time.Now()
		return nil

	case key.Matches(msg, m.keys.GotoBottom):
		m.viewport.GotoBottom()
		m.hexViewport.GotoBottom()
		m.refreshViewportContent()
		return nil

	case key.Matches(msg, m.keys.GoToLeft):
		m.viewport.GoToLeft()
		return nil

	case key.Matches(msg, m.keys.GoToRight):
		m.viewport.GoToRight()
		return nil

	case key.Matches(msg, m.keys.Left):
		m.viewport.ScrollLeft(3)
		m.refreshViewportContent()
		return nil

	case key.Matches(msg, m.keys.Right):
		m.viewport.ScrollRight(3)
		m.refreshViewportContent()
		return nil

	default:
		var cmd tea.Cmd
		m.hexViewport, cmd = m.hexViewport.Update(msg)
		m.viewport.SetYOffset(m.hexViewport.YOffset)
		m.refreshViewportContent()
		return cmd
	}
}

func (m *metricsModel) handleSearchInput(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.CancelSearch):
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.currentMatch = 0
			m.searchMatches = nil
			m.refreshViewportContent()
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
			m.findMatches(true)
		}
		return nil

	case msg.Type == tea.KeyRunes || msg.String() == " ":
		m.searchQuery += msg.String()
		m.findMatches(true)
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
