package loglist

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.enabled {
		return m, nil
	}

	switch msg := msg.(type) {
	case messages.HTTPLogReadyMsg:
		if msg.Log != nil {
			m.addLog(msg.Log)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.moveUp()

		case key.Matches(msg, m.keys.Down):
			m.moveDown()

		case key.Matches(msg, m.keys.GotoTop):
			if time.Since(m.lastGPress) < 500*time.Millisecond {
				m.gotoTop()
			}
			m.lastGPress = time.Now()

		case key.Matches(msg, m.keys.GotoBottom):
			m.gotoBottom()

		case key.Matches(msg, m.keys.Enter):
			if m.store.Len() > 0 {
				return m, func() tea.Msg {
					return messages.LogSelectedMsg{
						Log: m.store.Get(m.selectedIndex),
					}
				}
			}
		}
	}

	return m, nil
}
