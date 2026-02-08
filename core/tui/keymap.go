package tui

import "github.com/charmbracelet/bubbles/key"

type GlobalKeyMap struct {
	Quit key.Binding
	Back key.Binding
	Help key.Binding
}

type keyMapSlice []key.Binding

func (k keyMapSlice) ShortHelp() []key.Binding {
	return k
}

func (k keyMapSlice) FullHelp() [][]key.Binding {
	return [][]key.Binding{k}
}

func DefaultGlobalKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Quit: key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
		Back: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back, k.Quit}
}

func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Back, k.Quit, k.Help},
	}
}
