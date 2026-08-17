package tui

import "charm.land/bubbles/v2/key"

// KeyMap is every key mori listens for. It satisfies help.KeyMap, so the
// footer and the '?' screen are generated from exactly these bindings and
// can't drift away from what actually works.
//
// The organising idea is that horizontal moves through time and vertical
// moves through the page. That is why h and l are days rather than
// characters: in a one-column reading app, left and right have nothing else
// to mean.
type KeyMap struct {
	PrevDay key.Binding
	NextDay key.Binding
	Today   key.Binding
	Goto    key.Binding

	Up   key.Binding
	Down key.Binding
	Top  key.Binding
	End  key.Binding

	Write   key.Binding
	Section key.Binding
	Done    key.Binding
	Save    key.Binding

	Help   key.Binding
	Quit   key.Binding
	Cancel key.Binding
	Submit key.Binding
}

// DefaultKeys is keyboard-first and vim-shaped where vim has an opinion worth
// copying, with arrows for everyone else.
func DefaultKeys() KeyMap {
	return KeyMap{
		PrevDay: key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "previous day")),
		NextDay: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next day")),
		Today:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "today")),
		Goto:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "go to a date")),

		Up:   key.NewBinding(key.WithKeys("up", "k", "ctrl+p"), key.WithHelp("↑/k", "scroll up")),
		Down: key.NewBinding(key.WithKeys("down", "j", "ctrl+n"), key.WithHelp("↓/j", "scroll down")),
		Top:  key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "top")),
		End:  key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "bottom")),

		Write:   key.NewBinding(key.WithKeys("enter", "i"), key.WithHelp("↵", "write")),
		Section: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new section")),
		Done:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "done writing")),
		Save:    key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),

		Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keys")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+c")),
		Submit: key.NewBinding(key.WithKeys("enter")),
	}
}

// ShortHelp is the one-line footer.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.PrevDay, k.NextDay, k.Write, k.Today, k.Help, k.Quit}
}

// FullHelp is the '?' view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.PrevDay, k.NextDay, k.Today, k.Goto},
		{k.Up, k.Down, k.Top, k.End},
		{k.Write, k.Section, k.Done, k.Save},
		{k.Help, k.Quit},
	}
}

// writingHelp is the footer while you're actually writing, where most of the
// keys above belong to the text rather than to mori.
func (k KeyMap) writingHelp() []key.Binding {
	return []key.Binding{k.Done, k.Save}
}
