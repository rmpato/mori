package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/ui"
)

// refresh rebuilds everything the view needs, so View stays a pure assembly
// step with no hidden state changes.
func (m *Model) refresh() {
	if !m.ready {
		return
	}

	m.header = m.renderHeader()
	m.footer = m.renderFooter()

	height := m.height - lipgloss.Height(m.header) - lipgloss.Height(m.footer)
	if height < 1 {
		height = 1
	}

	if m.mode == modeWrite {
		m.ta.SetWidth(m.contentWidth())
		m.ta.SetHeight(height - 1) // the blank line under the rule
		return
	}

	m.vp.SetWidth(m.width)
	m.vp.SetHeight(height)
	m.vp.SetContentLines(m.renderPage())
}

// View assembles the screen.
func (m *Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	v.WindowTitle = "mori"
	if !m.ready {
		return v
	}

	middle := m.vp.View()
	if m.mode == modeWrite {
		middle = "\n" + indent(m.ta.View(), m.gutterWidth()+2)
	}

	frame := strings.Join([]string{m.header, middle, m.footer}, "\n")
	v.SetContent(m.clampFrame(frame))
	return v
}

// renderHeader is the season, the day, and — only when it is worth saying —
// where that day sits relative to now.
func (m *Model) renderHeader() string {
	glyph := ui.SeasonOf(m.date, m.south).Glyph()
	left := m.theme.Weekday.Render(glyph+"  "+m.date.Time().Format("Monday")) +
		m.theme.Date.Render(", "+m.date.Human())

	var right string
	switch rel := m.date.HumanRelative(m.now()); rel {
	case "today", "yesterday", "tomorrow":
		right = m.theme.Hint.Render(rel)
	}
	if m.page.Mood != "" {
		right = m.theme.Mood.Render(m.page.Mood) + "   " + right
	}

	return "\n" + m.gutter() + m.row(left, right) + "\n" + m.gutter() + m.theme.Line(m.contentWidth())
}

// renderFooter is the rule, then whatever mori has to say, then the keys.
func (m *Model) renderFooter() string {
	var b strings.Builder
	b.WriteString(m.gutter() + m.theme.Line(m.contentWidth()) + "\n")

	switch {
	case m.err != nil:
		b.WriteString(m.gutter() + m.theme.Warn.Render(m.fit(m.err.Error())) + "\n")
	case m.status != "":
		b.WriteString(m.gutter() + m.theme.Hint.Render(m.fit(m.status)) + "\n")
	case m.greeting != "" && m.mode == modeRead:
		b.WriteString(m.gutter() + m.theme.Aside.Render(m.fit(m.greeting)) + "\n")
	default:
		b.WriteString("\n")
	}

	m.help.SetWidth(m.contentWidth())
	switch m.mode {
	case modeWrite:
		b.WriteString(m.gutter() + m.help.ShortHelpView(m.keys.writingHelp()))
	case modeGoto:
		prompt := m.theme.Prompt.Render("go to  ")
		m.input.SetWidth(max(1, m.contentWidth()-ansi.StringWidth(prompt)))
		b.WriteString(m.gutter() + prompt + m.input.View())
	default:
		if m.help.ShowAll {
			// The full help is several lines, and every one of them needs the
			// gutter — not just the first.
			b.WriteString(indent(m.help.FullHelpView(m.keys.FullHelp()), m.gutterWidth()))
		} else {
			b.WriteString(m.gutter() + m.help.ShortHelpView(m.keys.ShortHelp()))
		}
	}
	return b.String()
}

// renderPage lays the day out for reading: wrapped prose, section headings
// shown as the times they are, and nothing else.
//
// An empty day renders as an empty page. There is no prompt, no invitation,
// and no remark about the blankness — that is what a fresh page looks like.
func (m *Model) renderPage() []string {
	if m.page.IsEmpty() {
		return []string{""}
	}

	width := m.contentWidth() - 2
	if width < 1 {
		width = 1
	}
	pad := m.gutter() + "  "

	out := []string{""}
	blank := true
	add := func(s string) {
		if s == "" {
			if !blank {
				out = append(out, "")
				blank = true
			}
			return
		}
		out = append(out, s)
		blank = false
	}

	for raw := range strings.Lines(m.page.Body) {
		raw = strings.TrimRight(raw, "\r\n")
		if at, ok := entry.SectionAt(raw); ok {
			add("")
			add(pad + m.theme.Time.Render(at))
			add("")
			continue
		}
		if strings.TrimSpace(raw) == "" {
			add("")
			continue
		}
		for _, line := range strings.Split(wrap(raw, width), "\n") {
			out = append(out, pad+m.theme.Body.Render(line))
			blank = false
		}
	}
	return out
}

// wrap breaks a line of prose to a width, on word boundaries where it can and
// mid-word where it must.
func wrap(s string, width int) string {
	return ansi.Hardwrap(ansi.Wordwrap(s, width, ""), width, true)
}

// indent shifts a block of already-rendered lines to the right.
func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

// contentWidth is mori's column: narrow, and centred in whatever it's given.
func (m *Model) contentWidth() int {
	w := m.width - 4
	if w > maxContentWidth {
		w = maxContentWidth
	}
	if w < minContentWidth {
		w = minContentWidth
	}
	if w > m.width {
		w = m.width
	}
	return w
}

func (m *Model) gutterWidth() int {
	g := (m.width - m.contentWidth()) / 2
	if g < 0 {
		return 0
	}
	return g
}

func (m *Model) gutter() string { return strings.Repeat(" ", m.gutterWidth()) }

// row lays out a left and a right chunk with the gap between them filled.
func (m *Model) row(left, right string) string {
	w := m.contentWidth()
	lw, rw := ansi.StringWidth(left), ansi.StringWidth(right)
	if lw+rw+1 > w {
		return ansi.Truncate(left, w, "…")
	}
	return left + strings.Repeat(" ", w-lw-rw) + right
}

// fit trims free text to mori's column, keeping its escape sequences intact.
func (m *Model) fit(s string) string { return ansi.Truncate(s, m.contentWidth(), "…") }

// clampFrame guarantees the frame fits the terminal. The renderers above size
// themselves properly, but a terminal can always be smaller than any layout
// is willing to be, and a frame that overflows corrupts the display — so this
// is the backstop that makes the invariant unconditional.
func (m *Model) clampFrame(frame string) string {
	lines := strings.Split(frame, "\n")
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	for i, l := range lines {
		if ansi.StringWidth(l) > m.width {
			lines[i] = ansi.Truncate(l, m.width, "")
		}
	}
	return strings.Join(lines, "\n")
}
