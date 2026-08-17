package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/rmpato/mori/internal/entry"
)

// The calendar is the one place mori shows you a shape rather than a page:
// which days you wrote on, laid out the way a month looks. It counts nothing
// and scores nothing — a month with three marks in it is a month with three
// marks in it.

// openCalendar shows the month the current day falls in, with the cursor on
// that day.
func (m *Model) openCalendar() {
	m.mode = modeCalendar
	m.cursor = m.date
	m.loadMonth()
}

// loadMonth reads which days in the cursor's month have pages. It is one
// readdir, which is exactly what the year/month layout is for.
func (m *Model) loadMonth() {
	first := m.cursor.FirstOfMonth()
	dates, err := m.store.Dates(first, first.LastOfMonth())
	if err != nil {
		m.err = err
		return
	}
	m.written = make(map[entry.Date]bool, len(dates))
	for _, d := range dates {
		m.written[d] = true
	}
	m.month = first
}

// moveCursor shifts the calendar cursor, loading a new month when it walks
// off the edge of this one.
func (m *Model) moveCursor(days int) {
	m.cursor = m.cursor.Add(days)
	if !m.cursor.SameMonth(m.month) {
		m.loadMonth()
	}
}

// showMonth jumps whole months, keeping the cursor on a day that exists —
// the 31st of a 30-day month lands on the 30th rather than spilling over.
func (m *Model) showMonth(delta int) {
	first := m.month.AddMonths(delta)
	day := m.cursor.Day
	if last := first.LastOfMonth().Day; day > last {
		day = last
	}
	m.cursor = entry.Date{Year: first.Year, Month: first.Month, Day: day}
	m.loadMonth()
}

// renderCalendar draws the month grid.
func (m *Model) renderCalendar() []string {
	first := m.month
	title := first.Time().Format("January 2006")

	// Weeks start on Monday, which is how a week is shaped when you're
	// looking back at one.
	lead := (int(first.Weekday()) + 6) % 7

	head := m.theme.Hint.Render("Mo Tu We Th Fr Sa Su")
	rows := []string{
		"",
		m.center(m.theme.Section.Render(title), 20),
		"",
		m.center(head, 20),
	}

	var week []string
	for range lead {
		week = append(week, "  ")
	}

	today := entry.DateOf(m.now())
	for day := 1; day <= first.LastOfMonth().Day; day++ {
		d := entry.Date{Year: first.Year, Month: first.Month, Day: day}
		week = append(week, m.renderDay(d, today))
		if len(week) == 7 {
			rows = append(rows, m.center(strings.Join(week, " "), 20))
			week = nil
		}
	}
	if len(week) > 0 {
		for len(week) < 7 {
			week = append(week, "  ")
		}
		rows = append(rows, m.center(strings.Join(week, " "), 20))
	}

	rows = append(rows, "", m.center(m.theme.Hint.Render("● a page"), 20))

	pad := m.gutter() + "  "
	for i, r := range rows {
		if r != "" {
			rows[i] = pad + r
		}
	}
	return rows
}

// renderDay is one cell: the number, said quietly or said clearly depending
// on whether there is anything behind it.
func (m *Model) renderDay(d, today entry.Date) string {
	label := fmt.Sprintf("%2d", d.Day)

	style := m.theme.Hint
	if m.written[d] {
		style = lipgloss.NewStyle().Foreground(m.theme.Brand).Bold(true)
	}
	if d == today {
		style = style.Underline(true)
	}
	if d == m.cursor {
		style = style.Reverse(true)
	}
	return style.Render(label)
}

// center pads a rendered string to sit in the middle of mori's column.
func (m *Model) center(s string, width int) string {
	left := (m.contentWidth() - width) / 2
	if left < 0 {
		left = 0
	}
	return strings.Repeat(" ", left) + s
}

// handleCalendarKey drives the month view.
func (m *Model) handleCalendarKey(msg tea.KeyPressMsg) bool {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Calendar):
		m.mode = modeRead

	case key.Matches(msg, m.keys.Select):
		m.mode = modeRead
		m.goTo(m.cursor)

	case key.Matches(msg, m.keys.PrevDay):
		m.moveCursor(-1)
	case key.Matches(msg, m.keys.NextDay):
		m.moveCursor(1)
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-7)
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(7)

	case key.Matches(msg, m.keys.PrevMonth):
		m.showMonth(-1)
	case key.Matches(msg, m.keys.NextMonth):
		m.showMonth(1)

	case key.Matches(msg, m.keys.Today):
		m.cursor = entry.DateOf(m.now())
		m.loadMonth()

	default:
		return false
	}
	return true
}
