package tui

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/facts"
	"github.com/rmpato/mori/internal/store"
	"github.com/rmpato/mori/internal/ui"
)

// Run opens mori on a day and returns once you leave it.
func Run(s *store.Store, d entry.Date, src facts.Source, out io.Writer) error {
	m := New(s, d, src)

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}

	fm, ok := final.(*Model)
	if !ok {
		return nil
	}
	if fm.Err() != nil {
		return fm.Err()
	}
	farewell(out, fm)
	return nil
}

// farewell leaves one quiet line in your scrollback: the day you were on, and
// nothing else. No count, no streak, no remark about how much you wrote.
func farewell(out io.Writer, m *Model) {
	line := lipgloss.NewStyle().Foreground(m.theme.Brand).Render(ui.FaceCalm) +
		lipgloss.NewStyle().Foreground(m.theme.Muted).Render("  "+m.date.Human())
	fmt.Fprintln(out, line)
}
