package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/search"
)

// Searching runs as you type, in the background, so a long journal never
// makes the prompt stutter. Only the newest answer is used: every keystroke
// takes a ticket, and a result whose ticket has been superseded is dropped.

// How many days a search in the interface will show. Anything more than this
// is a question that wants narrowing rather than scrolling.
const maxResults = 100

type searchDoneMsg struct {
	seq     int
	matches []search.Match
	err     error
}

type tagsDoneMsg struct {
	counts []search.TagCount
	err    error
}

// openSearch puts the cursor in the search prompt, keeping whatever was
// searched for last so refining a search doesn't mean retyping it.
func (m *Model) openSearch(query string) tea.Cmd {
	m.mode = modeSearch
	m.input.SetValue(query)
	m.input.CursorEnd()
	m.results = nil
	m.selected = 0
	return tea.Batch(m.input.Focus(), m.runSearch())
}

// runSearch answers the prompt as it currently reads.
func (m *Model) runSearch() tea.Cmd {
	m.findSeq++
	seq := m.findSeq
	text := strings.TrimSpace(m.input.Value())
	now := m.now()
	st := m.store

	if text == "" {
		m.results = nil
		return nil
	}

	return func() tea.Msg {
		q, err := search.Parse(text, now)
		if err != nil {
			return searchDoneMsg{seq: seq, err: err}
		}
		matches, err := search.All(st, q, maxResults)
		return searchDoneMsg{seq: seq, matches: matches, err: err}
	}
}

// openTags shows the tags you've used.
func (m *Model) openTags() tea.Cmd {
	m.mode = modeTags
	m.selected = 0
	st := m.store
	return func() tea.Msg {
		counts, err := search.Tags(st, entry.Date{}, entry.Date{})
		return tagsDoneMsg{counts: counts, err: err}
	}
}

// handleSearchKey drives the prompt and the list under it at the same time:
// the letters go to the prompt, the arrows to the results.
func (m *Model) handleSearchKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.mode = modeRead
		m.input.Blur()
		return nil

	case key.Matches(msg, m.keys.Up):
		m.move(-1, len(m.results))
		return nil
	case key.Matches(msg, m.keys.Down):
		m.move(1, len(m.results))
		return nil

	case key.Matches(msg, m.keys.Select):
		if m.selected < len(m.results) {
			d := m.results[m.selected].Entry.Date
			m.mode = modeRead
			m.input.Blur()
			m.goTo(d)
		}
		return nil
	}

	in, cmd := m.input.Update(msg)
	m.input = in
	return tea.Batch(cmd, m.runSearch())
}

// handleTagsKey drives the tag list. Choosing a tag is choosing a search.
func (m *Model) handleTagsKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Tags):
		m.mode = modeRead
		return nil

	case key.Matches(msg, m.keys.Up):
		m.move(-1, len(m.tags))
		return nil
	case key.Matches(msg, m.keys.Down):
		m.move(1, len(m.tags))
		return nil

	case key.Matches(msg, m.keys.Select):
		if m.selected < len(m.tags) {
			return m.openSearch("#" + m.tags[m.selected].Tag)
		}
	}
	return nil
}

// move walks a selection without falling off either end.
func (m *Model) move(delta, length int) {
	if length == 0 {
		m.selected = 0
		return
	}
	m.selected += delta
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= length {
		m.selected = length - 1
	}
}

// renderResults lists the days a search found: the date, then the line the
// words were actually on, with the words picked out.
func (m *Model) renderResults() []string {
	if strings.TrimSpace(m.input.Value()) == "" {
		return []string{""}
	}
	if len(m.results) == 0 {
		return []string{"", m.gutter() + "  " + m.theme.Aside.Render("nothing.")}
	}

	pad := m.gutter() + "  "
	width := m.contentWidth() - 12
	if width < 8 {
		width = 8
	}

	out := []string{""}
	for i, r := range m.results {
		mark := "  "
		date := m.theme.Date.Render(r.Entry.Date.Short())
		if i == m.selected {
			mark = m.theme.Weekday.Render("▸ ")
			date = m.theme.Weekday.Render(r.Entry.Date.Short())
		}
		line := ansi.Truncate(m.highlight(r.Excerpt, r.Hit), width, "…")
		out = append(out, pad+mark+padTo(date, 7)+"  "+line)
	}
	if len(m.results) == maxResults {
		out = append(out, "", pad+"  "+m.theme.Hint.Render("…and more. Narrow it down?"))
	}
	return out
}

// renderTags lists the tags, and how many days carry each one. The number is
// there because it is interesting, not because it is a score.
func (m *Model) renderTags() []string {
	if len(m.tags) == 0 {
		return []string{"", m.gutter() + "  " + m.theme.Aside.Render("no tags yet.")}
	}

	width := 0
	for _, c := range m.tags {
		if n := ansi.StringWidth(c.Tag) + 1; n > width {
			width = n
		}
	}

	pad := m.gutter() + "  "
	out := []string{""}
	for i, c := range m.tags {
		mark := "  "
		tag := m.theme.Tag.Render("#" + c.Tag)
		if i == m.selected {
			mark = m.theme.Weekday.Render("▸ ")
		}
		out = append(out, pad+mark+padTo(tag, width)+"  "+
			m.theme.Hint.Render(plural(c.Days, "day", "days")))
	}
	return out
}

// highlight picks the matched words out of the line they were found on.
func (m *Model) highlight(line, hit string) string {
	if hit == "" {
		return m.theme.Body.Render(line)
	}
	i := strings.Index(line, hit)
	if i < 0 {
		return m.theme.Body.Render(line)
	}
	return m.theme.Body.Render(line[:i]) +
		m.theme.Match.Render(hit) +
		m.theme.Body.Render(line[i+len(hit):])
}

// padTo right-fills a rendered string to a display width.
func padTo(s string, w int) string {
	if n := ansi.StringWidth(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
