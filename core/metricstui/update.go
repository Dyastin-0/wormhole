package metricstui

import (
	"fmt"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/stream"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
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
		m.viewport = viewport.New(width, msg.Height-verticalMarginHeight)
		m.viewport.YPosition = headerHeight
		m.hexViewport = viewport.New(width, msg.Height-verticalMarginHeight)
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

	switch msg.String() {
	case "ctrl+c", "q":
		return tea.Quit

	case "esc":
		if m.viewMode == ViewModeDetail {
			m.viewMode = ViewModeList
			return nil
		}
	}

	if m.viewMode == ViewModeDetail {
		switch msg.String() {
		case "l", "tab", "h", "shift+tab":
			m.activeTab = (m.activeTab + 1) % 2
			m.viewMode = ViewModeDetail
			m.setSelectedLog()
			return nil
		}
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
	switch msg.String() {
	case "enter":
		if m.logStore.Len() > 0 {
			m.setSelectedLog()
			m.currentMatch = 0
			m.viewMode = ViewModeDetail
			m.activeTab = TabResponseBody
		}

	case "up", "k":
		m.logStore.MoveUp()

	case "down", "j":
		m.logStore.MoveDown()
	}

	return nil
}

func (m *metricsModel) handleDetailViewKeys(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if m.searchMode {
		switch key {
		case "esc":
			if m.searchQuery != "" {
				m.currentMatch = 0
				m.searchQuery = ""
			} else {
				m.searchMode = false
			}
			return nil

		case "enter":
			if len(m.searchMatches) > 0 {
				m.currentMatch = 0
				m.jumpToMatch()
			}
			m.searchMode = false
			return nil

		case "backspace", "ctrl+h":
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				m.findMatches(true)
			}
			return nil

		default:
			if len(key) == 1 {
				m.searchQuery += key
				m.findMatches(true)
			}
			return nil
		}
	}

	switch key {
	case "/":
		m.searchMode = true
		return nil

	case "n":
		if len(m.searchMatches) > 0 {
			m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
			m.jumpToMatch()
		}
		return nil

	case "N":
		if len(m.searchMatches) > 0 {
			m.currentMatch = (m.currentMatch - 1 + len(m.searchMatches)) % len(m.searchMatches)
			m.jumpToMatch()
		}
		return nil

	case "g":
		if time.Since(m.lastGPress) < 500*time.Millisecond {
			m.viewport.GotoTop()
			m.hexViewport.GotoTop()
			m.lastGPress = time.Time{}
			m.refreshViewportContent()
			return nil
		}
		m.lastGPress = time.Now()
		return nil

	case "G":
		m.viewport.GotoBottom()
		m.hexViewport.GotoBottom()
		m.refreshViewportContent()
		m.lastGPress = time.Time{}
		return nil

	case "left":
		m.viewport.ScrollLeft(3)
		return nil
	case "right":
		m.viewport.ScrollRight(3)
		return nil
	default:
		m.lastGPress = time.Time{}
		var cmd tea.Cmd
		m.hexViewport, cmd = m.hexViewport.Update(msg)
		m.viewport.SetYOffset(m.hexViewport.YOffset)
		m.refreshViewportContent()
		return cmd
	}
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
