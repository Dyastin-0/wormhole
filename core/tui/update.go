package tui

import (
	"github.com/Dyastin-0/wormhole/core/tui/messages"
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
		m.logDetail, cmd = m.logDetail.Update(tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height,
		})
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) && !m.logDetail.IsSearching() {
			return m, tea.Quit
		}
		if key.Matches(msg, m.keys.Back) && m.viewMode == messages.ViewModeDetail {
			if !m.logDetail.IsSearching() {
				m.viewMode = messages.ViewModeList
				return m, nil
			}
		}

	case messages.MetricsMsg:
		m.metrics, cmd = m.metrics.Update(msg)
		return m, cmd

	case messages.HTTPLogMsg:
		m.logList, cmd = m.logList.Update(msg)
		return m, cmd

	case messages.LogSelectedMsg:
		m.viewMode = messages.ViewModeDetail
		m.logDetail, cmd = m.logDetail.Update(messages.SetLogMsg(msg))
		cmds = append(cmds, cmd)
		if m.width > 0 && m.height > 0 {
			var sizeCmd tea.Cmd
			m.logDetail, sizeCmd = m.logDetail.Update(tea.WindowSizeMsg{
				Width:  m.width,
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
