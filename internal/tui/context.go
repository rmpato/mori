package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/facts"
)

// The context pane is the whole of mori's relationship with tuki: a list of
// what you got done and what you didn't, sitting beside a blank page while you
// decide what to say about it.
//
// It is never more than that. Pressing ↵ lays the facts out as headings and
// bullets to write under — mori will not turn them into sentences, because
// then they wouldn't be yours.

type contextDoneMsg struct {
	date entry.Date
	snap facts.Snapshot
	err  error
}

// openContext shows what else is known about the day on screen.
func (m *Model) openContext() tea.Cmd {
	if m.source == nil {
		// No tuki, no context, no complaint. The key simply does nothing.
		return nil
	}
	m.mode = modeContext
	m.snap = facts.Snapshot{}

	src, d := m.source, m.date
	return func() tea.Msg {
		snap, err := src.Day(d)
		return contextDoneMsg{date: d, snap: snap, err: err}
	}
}

// handleContextKey drives the pane.
func (m *Model) handleContextKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Context):
		m.mode = modeRead
		return nil

	case key.Matches(msg, m.keys.Select):
		return m.startFromContext()
	}
	return nil
}

// startFromContext lays the day's facts into the page as something to write
// under, and puts the cursor at the end of it.
//
// It refuses over the top of writing. A starting point is only a starting
// point when there is nothing there yet.
func (m *Model) startFromContext() tea.Cmd {
	if m.snap.IsEmpty() {
		m.mode = modeRead
		return nil
	}
	if !m.page.IsEmpty() {
		m.setStatus("there's already writing on this page")
		return m.expireStatus()
	}

	m.page.Body = strings.TrimRight(facts.Template(m.date, m.snap), "\n")
	m.commit()
	m.mode = modeRead
	return m.beginWrite(false)
}

// renderContext lists what tuki knows: what got done, and what didn't.
func (m *Model) renderContext() []string {
	pad := m.gutter() + "  "

	if m.snap.IsEmpty() {
		return []string{"", pad + m.theme.Aside.Render("tuki has nothing for this day.")}
	}

	out := []string{""}
	section := func(mark, label string, items []facts.Item, style func(...string) string) {
		if len(items) == 0 {
			return
		}
		out = append(out, pad+m.theme.Hint.Render(label), "")
		for _, it := range items {
			line := pad + "  " + style(mark) + " " + m.theme.Body.Render(it.Text)
			if it.Tag != "" {
				line += "  " + m.theme.Tag.Render("#"+it.Tag)
			}
			out = append(out, m.fit(line))
		}
		out = append(out, "")
	}

	section("✓", "you finished", m.snap.Done, m.theme.Tag.Render)
	section("○", "still open", m.snap.Todo, m.theme.Hint.Render)

	if m.page.IsEmpty() {
		out = append(out, pad+m.theme.Aside.Render("↵ to start a page from this."))
	}
	return out
}
