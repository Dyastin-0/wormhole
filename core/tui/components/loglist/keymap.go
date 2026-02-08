package loglist

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view")),
	}
}

func (k KeyMap) ShortKey() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter}
}
