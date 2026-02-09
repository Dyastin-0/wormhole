package tui

import (
	"fmt"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/tui/formatters"
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/Dyastin-0/wormhole/stream"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.metrics, cmd = m.metrics.Update(msg)
		cmds = append(cmds, cmd)
		m.logList, cmd = m.logList.Update(msg)
		cmds = append(cmds, cmd)

		leftPanelWidth := styles.HeaderKeyWidth + /*header colon*/ 1 + styles.HeaderValueWidth + /*column padding*/ 2
		detailWidth := msg.Width - leftPanelWidth
		m.logDetail, cmd = m.logDetail.Update(tea.WindowSizeMsg{
			Width:  detailWidth,
			Height: msg.Height,
		})
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		if key.Matches(msg, m.keys.Back) && m.viewMode == messages.ViewModeDetail {
			if !m.logDetail.IsSearching() {
				m.viewMode = messages.ViewModeList
				return m, nil
			}
		}

		// if key.Matches(msg, m.keys.Help) {
		// 	if m.viewMode != messages.ViewModeDetail {
		// 		m.help.ShowAll = !m.help.ShowAll
		// 		return m, nil
		// 	}
		// }

	case messages.MetricsMsg:
		m.metrics, cmd = m.metrics.Update(msg)
		return m, cmd

	case *stream.Response:
		return m, m.handleHTTPResponse(msg)

	case messages.HTTPLogReadyMsg:
		m.logList, cmd = m.logList.Update(msg)
		return m, cmd

	case messages.LogSelectedMsg:
		m.viewMode = messages.ViewModeDetail
		m.logDetail, cmd = m.logDetail.Update(messages.SetLogMsg(msg))
		cmds = append(cmds, cmd)

		if m.width > 0 && m.height > 0 {
			leftPanelWidth := styles.HeaderKeyWidth + /*header colon*/ 1 + styles.HeaderValueWidth + /*column padding*/ 2
			detailWidth := m.width - leftPanelWidth

			var sizeCmd tea.Cmd
			m.logDetail, sizeCmd = m.logDetail.Update(tea.WindowSizeMsg{
				Width:  detailWidth,
				Height: m.height,
			})
			cmds = append(cmds, sizeCmd)
		}

		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.viewMode {
	case messages.ViewModeList:
		m.logList, cmd = m.logList.Update(msg)
		cmds = append(cmds, cmd)

	case messages.ViewModeDetail:
		m.logDetail, cmd = m.logDetail.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleHTTPResponse(msg *stream.Response) tea.Cmd {
	log := &messages.HTTPLogMsg{Response: msg}

	responseBody, err := formatters.ReadResponseBody(msg)
	if err != nil {
		responseBody = fmt.Appendf(nil, "Error reading body: %v", err)
	}
	log.ResponseBody = responseBody

	return func() tea.Msg {
		var protoLog *proto.HTTPLog

		select {
		case request := <-m.requestch:
			log.Request = request
			requestBody, err := formatters.ReadRequestBody(request)
			if err != nil {
				requestBody = fmt.Appendf(nil, "Error reading body: %v", err)
			}
			log.RequestBody = requestBody
		case <-time.After(5 * time.Second):
		}

		select {
		case protoLog = <-m.httpLogch:
		case <-time.After(5 * time.Second):
			return nil
		}

		log.HTTPLog = protoLog

		return messages.HTTPLogReadyMsg{Log: log}
	}
}
