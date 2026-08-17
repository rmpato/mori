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
	Editor  key.Binding

	Calendar  key.Binding
	Search    key.Binding
	Tags      key.Binding
	Mood      key.Binding
	Context   key.Binding
	PrevMonth key.Binding
	NextMonth key.Binding
	Select    key.Binding
	Back      key.Binding

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
		Done:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "done")),
		Save:    key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
		Editor:  key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "$EDITOR")),

		Calendar:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "calendar")),
		Search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Tags:      key.NewBinding(key.WithKeys("#"), key.WithHelp("#", "tags")),
		Mood:      key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mood")),
		Context:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "context")),
		PrevMonth: key.NewBinding(key.WithKeys("[", ","), key.WithHelp("[", "previous month")),
		NextMonth: key.NewBinding(key.WithKeys("]", "."), key.WithHelp("]", "next month")),
		Select:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "open")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),

		Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keys")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+c")),
		Submit: key.NewBinding(key.WithKeys("enter")),
	}
}

// ShortHelp is the one-line footer.
//
// The day keys are described as one entry rather than two. Spelled out in
// full — "←/h previous day • →/l next day" — the line comes to 83 characters,
// which does not fit in mori's 76-column page on any terminal, so the last
// thing in it silently loses its tail.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		relabel(k.PrevDay, "←→", "day"),
		k.Write,
		k.Calendar,
		k.Search,
		k.Help,
		k.Quit,
	}
}

// FullHelp is the '?' view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.PrevDay, k.NextDay, k.Today, k.Goto},
		{k.Up, k.Down, k.Top, k.End},
		{k.Write, k.Section, k.Done, k.Save, k.Editor, k.Mood},
		{k.Calendar, k.Search, k.Tags, k.Context, k.Help, k.Quit},
	}
}

// writingHelp is the footer while you're actually writing, where most of the
// keys above belong to the text rather than to mori.
func (k KeyMap) writingHelp() []key.Binding {
	return []key.Binding{relabel(k.Done, "esc", "done writing"), k.Save}
}

// calendarHelp is the footer in the month view, where up and down are weeks
// rather than lines.
func (k KeyMap) calendarHelp() []key.Binding {
	return []key.Binding{
		relabel(k.PrevDay, "←→", "day"),
		relabel(k.Up, "↑↓", "week"),
		relabel(k.PrevMonth, "[ ]", "month"),
		k.Select,
		k.Back,
	}
}

// contextHelp is the footer beside what tuki knows.
func (k KeyMap) contextHelp() []key.Binding {
	return []key.Binding{relabel(k.Select, "↵", "start a page from this"), k.Back}
}

// listHelp is the footer under a list of results or tags.
func (k KeyMap) listHelp() []key.Binding {
	return []key.Binding{relabel(k.Up, "↑↓", "move"), k.Select, k.Back}
}

// relabel says what a key does in the mode you're in — up and down are lines
// in a page, weeks in a month, and results in a list.
//
// It copies the binding's real keys rather than restating them, so a footer
// can describe a key differently but can never describe a key that isn't
// there.
func relabel(b key.Binding, keys, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(b.Keys()...), key.WithHelp(keys, desc))
}
