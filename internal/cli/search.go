package cli

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/search"
)

func newSearchCmd() *cobra.Command {
	var (
		plain  bool
		asJSON bool
		limit  int
	)
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "look for something you wrote",
		Long: `Look for something you wrote.

  mori search photography
  mori search "the zine idea"
  mori search go project          both words, in any order
  mori search "#photography"      a tag
  mori search gym since:2026-01-01

Bare words match the start of a word, so "photo" finds "photography". A
quoted phrase is matched exactly.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := newEnv(cmd)
			if err != nil {
				return err
			}
			q, err := search.Parse(strings.Join(args, " "), e.now)
			if err != nil {
				return err
			}

			matches, err := search.All(e.store, q, limit)
			if err != nil {
				return err
			}

			if asJSON {
				out := make([]matchJSON, 0, len(matches))
				for _, m := range matches {
					out = append(out, matchJSON{
						Date:    m.Entry.Date.String(),
						Excerpt: m.Excerpt,
						Tags:    m.Entry.Tags(),
						Mood:    m.Entry.Mood,
					})
				}
				return writeJSON(e.out, out)
			}

			if len(matches) == 0 {
				if e.tty {
					e.println(e.theme.Aside.Render("nothing."))
				}
				return nil
			}
			for _, m := range matches {
				e.println(e.matchLine(m, plain || !e.tty))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "tab-separated, for scripts")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the matches as JSON")
	cmd.Flags().IntVarP(&limit, "limit", "n", 0, "stop after this many days (0 for all)")
	return cmd
}

type matchJSON struct {
	Date    string   `json:"date"`
	Excerpt string   `json:"excerpt"`
	Tags    []string `json:"tags,omitempty"`
	Mood    string   `json:"mood,omitempty"`
}

// matchLine is one result: the day, then the line it was found on, with the
// hit picked out.
func (e *env) matchLine(m search.Match, plain bool) string {
	if plain {
		return m.Entry.Date.String() + "\t" + m.Excerpt
	}

	const gutter = 9 // the date column, plus the two spaces after it
	date := e.theme.Date.Render(pad(m.Entry.Date.Short(), 7))
	line := e.highlight(m.Excerpt, m.Hit)
	return date + "  " + ansi.Truncate(line, max(1, e.width()-gutter), "…")
}

// highlight picks the matched text out of the line it was found on.
func (e *env) highlight(line, hit string) string {
	if hit == "" {
		return e.theme.Body.Render(line)
	}
	i := strings.Index(line, hit)
	if i < 0 {
		return e.theme.Body.Render(line)
	}
	return e.theme.Body.Render(line[:i]) +
		e.theme.Match.Render(hit) +
		e.theme.Body.Render(line[i+len(hit):])
}

// pad right-fills to a display width, which is not the same as a byte count
// once a tag has an accent in it.
func pad(s string, w int) string {
	n := ansi.StringWidth(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

func newTagsCmd() *cobra.Command {
	var plain, asJSON bool
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "the tags you've used",
		Long: `The tags you've used, and how many days carry each one.

Tags are hashtags in the writing itself — there is nothing to set up and
nothing to maintain.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			e, err := newEnv(cmd)
			if err != nil {
				return err
			}
			counts, err := search.Tags(e.store, entry.Date{}, entry.Date{})
			if err != nil {
				return err
			}

			if asJSON {
				return writeJSON(e.out, counts)
			}
			if len(counts) == 0 {
				if e.tty {
					e.println(e.theme.Aside.Render("no tags yet."))
				}
				return nil
			}

			width := 0
			for _, c := range counts {
				if n := ansi.StringWidth(c.Tag); n > width {
					width = n
				}
			}
			for _, c := range counts {
				if plain || !e.tty {
					e.println(c.Tag + "\t" + strconv.Itoa(c.Days))
					continue
				}
				e.println(e.theme.Tag.Render(pad("#"+c.Tag, width+2)) + "  " +
					e.theme.Hint.Render(plural(c.Days, "day", "days")))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "tab-separated, for scripts")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the tags as JSON")
	return cmd
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

func newListCmd() *cobra.Command {
	var (
		plain, asJSON bool
		since         string
		limit         int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "the days you've written",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			e, err := newEnv(cmd)
			if err != nil {
				return err
			}

			var q search.Query
			if since != "" {
				d, err := entry.ParseDate(since, e.now)
				if err != nil {
					return err
				}
				q.Since = d
			}

			matches, err := search.All(e.store, q, limit)
			if err != nil {
				return err
			}

			if asJSON {
				out := make([]matchJSON, 0, len(matches))
				for _, m := range matches {
					out = append(out, matchJSON{
						Date:    m.Entry.Date.String(),
						Excerpt: m.Excerpt,
						Tags:    m.Entry.Tags(),
						Mood:    m.Entry.Mood,
					})
				}
				return writeJSON(e.out, out)
			}
			if len(matches) == 0 {
				if e.tty {
					e.println(e.theme.Aside.Render("nothing written yet."))
				}
				return nil
			}

			for _, m := range matches {
				if plain || !e.tty {
					e.println(m.Entry.Date.String() + "\t" + m.Excerpt)
					continue
				}
				e.println(e.theme.Date.Render(pad(m.Entry.Date.Short(), 7)) + "  " +
					e.theme.Body.Render(m.Entry.Excerpt(e.width()-9)))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "tab-separated, for scripts")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the days as JSON")
	cmd.Flags().StringVar(&since, "since", "", "only days from this one onwards")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "stop after this many days (0 for all)")
	return cmd
}
