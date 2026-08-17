// Package cli is mori's command-line half: the part that works in pipes,
// scripts, and one-liners, without opening the full-screen interface.
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/store"
	"github.com/rmpato/mori/internal/ui"
)

// env carries what every command needs: where the journal is, where output
// goes, and whether there is anyone actually looking at it.
type env struct {
	store *store.Store
	out   io.Writer
	w     io.Writer // colour-aware wrapper around out
	theme ui.Theme
	south bool
	tty   bool
	now   time.Time
}

// newEnv resolves the journal and works out how much decoration is warranted.
func newEnv(cmd *cobra.Command) (*env, error) {
	dir, _ := cmd.Flags().GetString("dir")

	var s *store.Store
	if dir != "" {
		s = store.New(dir)
	} else {
		var err error
		if s, err = store.Default(); err != nil {
			return nil, err
		}
	}

	out := cmd.OutOrStdout()
	return &env{
		store: s,
		out:   out,
		w:     colorprofile.NewWriter(out, os.Environ()),
		theme: ui.New(ui.PrefersDark()),
		south: ui.SouthernHemisphere(),
		tty:   isTerminal(out),
		now:   time.Now(),
	}, nil
}

// width is how wide the terminal is, or a comfortable default when nobody is
// looking.
func (e *env) width() int {
	f, ok := e.out.(*os.File)
	if !ok {
		return 80
	}
	w, _, err := term.GetSize(f.Fd())
	if err != nil || w < 20 {
		return 80
	}
	return w
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

func (e *env) printf(format string, a ...any) { fmt.Fprintf(e.w, format, a...) }
func (e *env) println(a ...any)               { fmt.Fprintln(e.w, a...) }

// today resolves the day a command is about: the argument if there is one,
// today if there isn't.
func (e *env) date(args []string) (entry.Date, error) {
	if len(args) == 0 {
		return entry.DateOf(e.now), nil
	}
	return entry.ParseDate(args[0], e.now)
}

// header is the line above a page: a season glyph, the date, and nothing else.
func (e *env) header(d entry.Date) string {
	glyph := ui.SeasonOf(d, e.south).Glyph()
	return e.theme.Weekday.Render(glyph+"  "+d.Time().Format("Monday")) +
		e.theme.Date.Render(", "+d.Human())
}

// writeEntry prints a page the way mori would want you to read it.
func (e *env) writeEntry(en entry.Entry, plain bool) {
	if plain {
		if !en.IsEmpty() {
			fmt.Fprint(e.out, string(en.Format()))
		}
		return
	}

	e.println(e.header(en.Date))
	if en.Mood != "" {
		e.println(e.theme.Mood.Render("   " + en.Mood))
	}
	e.println()

	if en.IsEmpty() {
		e.println(e.theme.Aside.Render("   nothing written here."))
		e.println()
		return
	}

	// blank tracks whether the last line written was empty, so the spacing
	// mori adds around a section heading never doubles up with the spacing
	// already in the page. Blank lines are written truly blank — a styled
	// run of spaces is invisible until someone selects it.
	blank := true
	line := func(s string) {
		if s == "" {
			if !blank {
				e.println()
				blank = true
			}
			return
		}
		e.println(s)
		blank = false
	}

	for raw := range strings.Lines(en.Body) {
		raw = strings.TrimRight(raw, "\r\n")
		if at, ok := entry.SectionAt(raw); ok {
			line("")
			line(e.theme.Time.Render("   " + at))
			line("")
			continue
		}
		if strings.TrimSpace(raw) == "" {
			line("")
			continue
		}
		line(e.theme.Body.Render("   " + raw))
	}
	e.println()
}

// entryJSON is the shape mori promises to scripts. It is flat on purpose:
// tags and sections are derived, but a caller shouldn't have to know that.
type entryJSON struct {
	Date     string          `json:"date"`
	Mood     string          `json:"mood,omitempty"`
	Body     string          `json:"body"`
	Tags     []string        `json:"tags,omitempty"`
	Words    int             `json:"words"`
	Sections []entry.Section `json:"sections,omitempty"`
}

func toJSON(en entry.Entry) entryJSON {
	return entryJSON{
		Date:     en.Date.String(),
		Mood:     en.Mood,
		Body:     en.Body,
		Tags:     en.Tags(),
		Words:    en.Words(),
		Sections: en.Sections(),
	}
}

// confirm asks a yes/no question, but only when there's a human to ask.
func confirm(e *env, in io.Reader, question string) (bool, error) {
	if !e.tty {
		return false, fmt.Errorf("%s — rerun with --yes", question)
	}
	e.printf("  %s %s ", e.theme.Prompt.Render(question), e.theme.Hint.Render("[y/N]"))
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
