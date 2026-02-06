package metricstui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit  key.Binding
	Back  key.Binding
	Tab   key.Binding
	Enter key.Binding

	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	GoToLeft   key.Binding
	GoToRight  key.Binding
	GotoTop    key.Binding
	GotoBottom key.Binding

	Search       key.Binding
	NextMatch    key.Binding
	PrevMatch    key.Binding
	CancelSearch key.Binding

	Help key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:  key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
		Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Tab:   key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "next tab")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),

		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
		Right:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
		GoToLeft:   key.NewBinding(key.WithKeys("shift+left", "H"), key.WithHelp("H", "left")),
		GoToRight:  key.NewBinding(key.WithKeys("shift+right", "L"), key.WithHelp("L", "right")),
		GotoTop:    key.NewBinding(key.WithKeys("g"), key.WithHelp("gg", "top")),
		GotoBottom: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),

		Search:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		NextMatch:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		PrevMatch:    key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
		CancelSearch: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel search")),

		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back, k.Enter, k.Search, k.NextMatch, k.PrevMatch, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.GotoTop, k.GotoBottom, k.GoToLeft, k.GoToRight},
		{k.Search, k.NextMatch, k.PrevMatch, k.CancelSearch},
		{k.Enter, k.Tab, k.Back, k.Quit},
	}
}
