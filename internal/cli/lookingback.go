package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/search"
	"github.com/rmpato/mori/internal/ui"
)

// Looking back is the long game: after a year of writing, a month should be
// able to tell you something about itself.
//
// What it must never become is a report card. So: counts of what happened,
// in the past tense, with no comparison to any other month, no average, no
// percentage, no target, and no adjective about whether it was enough. A
// month with two days in it prints two days and says nothing else about it.

// overview is a month, read back.
type overview struct {
	Month string            `json:"month"`
	Days  int               `json:"days"`
	First string            `json:"first,omitempty"`
	Last  string            `json:"last,omitempty"`
	Tags  []search.TagCount `json:"tags,omitempty"`
	Tuki  *tukiOverview     `json:"tuki,omitempty"`
}

type tukiOverview struct {
	Finished int       `json:"finished"`
	Tags     []tukiTag `json:"tags,omitempty"`
}

// tukiTag counts finished tasks, not days — the journal's own tag counts mean
// something different, and one field named for the other would make the JSON
// quietly wrong.
type tukiTag struct {
	Tag      string `json:"tag"`
	Finished int    `json:"finished"`
}

func newLookingBackCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "looking-back [month]",
		Aliases: []string{"back"},
		Short:   "a month, read back",
		Long: `Look back at a month.

  mori looking-back            this month
  mori looking-back "last month"
  mori looking-back august
  mori looking-back 2026-08

What you get is what happened: the days you wrote on, the tags you used, and —
if you use tuki — what you finished. There is nothing here that compares one
month to another, and nothing that tells you whether it was enough.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := newEnv(cmd)
			if err != nil {
				return err
			}

			arg := ""
			if len(args) > 0 {
				arg = args[0]
			}
			month, err := parseMonth(arg, e.now)
			if err != nil {
				return err
			}

			ov, err := e.lookBack(month)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(e.out, ov)
			}
			e.printOverview(month, ov)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the month as JSON")
	return cmd
}

// parseMonth reads a month the way you'd say one. It accepts everything
// ParseDate does and takes the month off it, plus a bare month name.
func parseMonth(s string, now time.Time) (entry.Date, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return entry.DateOf(now).FirstOfMonth(), nil
	}

	// A bare month name means the most recent one of those, which for the
	// current month is this one.
	if name, ok := monthByName(s); ok {
		d := entry.Date{Year: entry.DateOf(now).Year, Month: name, Day: 1}
		if d.After(entry.DateOf(now).FirstOfMonth()) {
			d.Year--
		}
		return d, nil
	}
	// "2026-08" is a month, not a day, and ParseDate would not know that.
	if t, err := time.Parse("2006-01", s); err == nil {
		return entry.Date{Year: t.Year(), Month: t.Month(), Day: 1}, nil
	}

	d, err := entry.ParseDate(s, now)
	if err != nil {
		return entry.Date{}, err
	}
	return d.FirstOfMonth(), nil
}

func monthByName(s string) (time.Month, bool) {
	for m := time.January; m <= time.December; m++ {
		full := strings.ToLower(m.String())
		if s == full || s == full[:3] {
			return m, true
		}
	}
	return 0, false
}

// lookBack gathers the month.
func (e *env) lookBack(month entry.Date) (overview, error) {
	first, last := month, month.LastOfMonth()

	dates, err := e.store.Dates(first, last)
	if err != nil {
		return overview{}, err
	}
	tags, err := search.Tags(e.store, first, last)
	if err != nil {
		return overview{}, err
	}

	ov := overview{
		Month: first.Time().Format("January 2006"),
		Days:  len(dates),
		Tags:  tags,
	}
	if len(dates) > 0 {
		ov.First = dates[0].String()
		ov.Last = dates[len(dates)-1].String()
	}

	// What tuki knows, if there is a tuki to ask. Reading it a day at a time
	// is cheap: the source keeps the file it parsed until it changes.
	if src, ok := e.tuki(); ok {
		counts := map[string]int{}
		finished := 0
		for d := first; !d.After(last); d = d.Add(1) {
			snap, err := src.Day(d)
			if err != nil {
				// tuki having a bad day is not mori's problem to make loud.
				counts, finished = nil, 0
				break
			}
			for _, item := range snap.Done {
				finished++
				if item.Tag != "" {
					counts[item.Tag]++
				}
			}
		}
		if finished > 0 {
			ov.Tuki = &tukiOverview{Finished: finished, Tags: sortCounts(counts)}
		}
	}
	return ov, nil
}

// sortCounts orders tags the way `mori tags` does: most first, then
// alphabetically.
func sortCounts(counts map[string]int) []tukiTag {
	out := make([]tukiTag, 0, len(counts))
	for tag, n := range counts {
		out = append(out, tukiTag{Tag: tag, Finished: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Finished != out[j].Finished {
			return out[i].Finished > out[j].Finished
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

// printOverview says what happened, and stops.
func (e *env) printOverview(month entry.Date, ov overview) {
	glyph := ui.SeasonOf(month, e.south).Glyph()

	e.println()
	e.println("  " + glyph + "  " + e.theme.Section.Render(ov.Month))
	e.println()

	if ov.Days == 0 {
		e.println("  " + e.theme.Aside.Render("Nothing written."))
		e.println()
		return
	}

	e.println("  " + e.theme.Body.Render(fmt.Sprintf("You wrote on %s.", plural(ov.Days, "day", "days"))))

	if len(ov.Tags) > 0 {
		e.println()
		width := 0
		for _, c := range ov.Tags {
			if n := len(c.Tag) + 1; n > width {
				width = n
			}
		}
		for _, c := range ov.Tags {
			e.println("  " + e.theme.Tag.Render(pad("#"+c.Tag, width)) + "   " +
				e.theme.Hint.Render(plural(c.Days, "day", "days")))
		}
	}

	if ov.Tuki != nil {
		e.println()
		line := fmt.Sprintf("In tuki, you finished %s.", plural(ov.Tuki.Finished, "thing", "things"))
		if len(ov.Tuki.Tags) > 0 {
			top := ov.Tuki.Tags[0]
			line = fmt.Sprintf("In tuki, you finished %s, %d of them #%s.",
				plural(ov.Tuki.Finished, "thing", "things"), top.Finished, top.Tag)
		}
		e.println("  " + e.theme.Body.Render(line))
	}
	e.println()
}
