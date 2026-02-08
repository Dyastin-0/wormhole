package logdetail

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Tab          key.Binding
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	GoToLeft     key.Binding
	GoToRight    key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	Search       key.Binding
	NextMatch    key.Binding
	PrevMatch    key.Binding
	CancelSearch key.Binding
	WrapText     key.Binding
	NormalCase   key.Binding
	Help         key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Tab:          key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "switch tab")),
		Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:         key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:        key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		GoToLeft:     key.NewBinding(key.WithKeys("shift+left", "H"), key.WithHelp("H", "start")),
		GoToRight:    key.NewBinding(key.WithKeys("shift+right", "L"), key.WithHelp("L", "end")),
		GotoTop:      key.NewBinding(key.WithKeys("g", "shift+up"), key.WithHelp("↑/gg", "top")),
		GotoBottom:   key.NewBinding(key.WithKeys("G", "shift+down"), key.WithHelp("↓/G", "bottom")),
		Search:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		NextMatch:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		PrevMatch:    key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
		CancelSearch: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		WrapText:     key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "wrap text")),
		NormalCase:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "normal case search")),
		Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Search, k.NextMatch, k.PrevMatch, k.Tab}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.GotoTop, k.GotoBottom, k.GoToLeft, k.GoToRight},
		{k.Search, k.NextMatch, k.PrevMatch, k.CancelSearch},
		{k.Tab, k.Help, k.WrapText, k.NormalCase},
	}
}
