// Package loglist implements the log list component.
package loglist

import (
	"time"

	"github.com/Dyastin-0/wormhole/core/tui/store"
	"github.com/Dyastin-0/wormhole/stream"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	maxLogs        = 1024
	minVisibleLogs = 10
)

type Model struct {
	enabled       bool
	store         *store.LogStore
	selectedIndex int
	scrollOffset  int
	visibleHeight int
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

func (m Model) GetSelected() *stream.HTTPEvent {
	return m.store.GetByIndex(m.selectedIndex)
}

func (m *Model) addLog(event *stream.HTTPEvent) {
	m.store.AddEvent(event)

	if m.selectedIndex == m.store.Len()-2 {
		m.selectedIndex = m.store.Len() - 1
	}

	if m.selectedIndex >= m.store.Len() {
		m.selectedIndex = m.store.Len() - 1
	}

	if m.selectedIndex >= m.scrollOffset+m.visibleHeight {
		m.scrollOffset = m.selectedIndex - m.visibleHeight + 1
	}
}

func (m *Model) moveUp() {
	if m.store.Len() == 0 {
		return
	}
	m.selectedIndex = (m.selectedIndex - 1 + m.store.Len()) % m.store.Len()
	if m.selectedIndex == m.store.Len()-1 {
		m.scrollOffset = max(0, m.store.Len()-m.visibleHeight)
	} else if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}
}

func (m *Model) moveDown() {
	if m.store.Len() == 0 {
		return
	}
	m.selectedIndex = (m.selectedIndex + 1) % m.store.Len()
	if m.selectedIndex == 0 {
		m.scrollOffset = 0
	} else if m.selectedIndex >= m.scrollOffset+m.visibleHeight {
		m.scrollOffset = m.selectedIndex - m.visibleHeight + 1
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
		if m.selectedIndex >= m.visibleHeight {
			m.scrollOffset = m.selectedIndex - m.visibleHeight + 1
		} else {
			m.scrollOffset = 0
		}
	}
}
