// Package loglist implements the log list component.
package loglist

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/store"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	maxLogs        = 1024
	maxVisibleLogs = 10
)

type Model struct {
	enabled       bool
	store         *store.LogStore
	selectedIndex int
	scrollOffset  int
	keys          KeyMap

	lastGPress time.Time
}

func New(enabled bool) Model {
	return Model{
		enabled:       enabled,
		store:         store.New(maxLogs),
		selectedIndex: 0,
		scrollOffset:  0,
		keys:          DefaultKeyMap(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Keys() KeyMap {
	return m.keys
}

func (m Model) GetSelected() *messages.HTTPLogMsg {
	return m.store.Get(m.selectedIndex)
}

func (m *Model) addLog(log *messages.HTTPLogMsg) {
	m.store.Add(log)

	if m.selectedIndex == m.store.Len()-2 {
		m.selectedIndex = m.store.Len() - 1
	}

	if m.selectedIndex >= m.store.Len() {
		m.selectedIndex = m.store.Len() - 1
	}

	if m.selectedIndex >= m.scrollOffset+maxVisibleLogs {
		m.scrollOffset = m.selectedIndex - maxVisibleLogs + 1
	}
}

func (m *Model) moveUp() {
	if m.store.Len() > 0 && m.selectedIndex > 0 {
		m.selectedIndex--
		if m.selectedIndex < m.scrollOffset {
			m.scrollOffset = m.selectedIndex
		}
	}
}

func (m *Model) moveDown() {
	if m.store.Len() > 0 && m.selectedIndex < m.store.Len()-1 {
		m.selectedIndex++
		if m.selectedIndex >= m.scrollOffset+maxVisibleLogs {
			m.scrollOffset = m.selectedIndex - maxVisibleLogs + 1
		}
	}
}

func (m *Model) gotoTop() {
	if m.store.Len() > 0 {
		m.selectedIndex = 0
		m.scrollOffset = 0
	}
}

func (m *Model) gotoBottom() {
	if m.store.Len() > 0 {
		m.selectedIndex = m.store.Len() - 1
		if m.selectedIndex >= maxVisibleLogs {
			m.scrollOffset = m.selectedIndex - maxVisibleLogs + 1
		} else {
			m.scrollOffset = 0
		}
	}
}
