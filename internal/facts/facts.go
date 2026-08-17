// Package facts answers "what else is known about this day?" from outside
// mori's own journal.
//
// Today there is one source: tuki, mori's sibling, which keeps the things you
// meant to do. The relationship is deliberately one-directional and
// deliberately thin — tuki provides the facts, mori provides the reflection,
// and mori never writes a sentence and attributes it to you.
package facts

import (
	"github.com/rmpato/mori/internal/entry"
)

// Item is one thing that was on your list.
type Item struct {
	Text string `json:"text"`
	Tag  string `json:"tag,omitempty"`
}

// Snapshot is what another tool knows about a day.
type Snapshot struct {
	Source string `json:"source"`
	Done   []Item `json:"done,omitempty"`
	Todo   []Item `json:"todo,omitempty"`
}

// IsEmpty reports whether there is nothing to say about the day.
func (s Snapshot) IsEmpty() bool { return len(s.Done) == 0 && len(s.Todo) == 0 }

// Source is something that can say what else happened on a day.
//
// It is an interface with one implementation because the shape matters more
// than the count: a git log, a calendar, or a music player would all fit here
// without the interface above it learning anything new.
type Source interface {
	Day(d entry.Date) (Snapshot, error)
}

// Template turns a snapshot into the start of a page: headings, and the facts
// under them, and empty space to write in.
//
// It is a scaffold and nothing more. mori will not write prose from your task
// list — the whole point of the two tools is that one of them holds what you
// did and the other holds what it was like, and only you can supply the
// second.
func Template(d entry.Date, s Snapshot) string {
	var b []string
	add := func(lines ...string) { b = append(b, lines...) }

	add("# " + d.Human())
	add("", "## Today", "")

	if len(s.Done) > 0 {
		add("## Things I did", "")
		for _, it := range s.Done {
			add("- " + it.Text)
		}
		add("")
	}
	if len(s.Todo) > 0 {
		add("## Things I didn't get to", "")
		for _, it := range s.Todo {
			add("- " + it.Text)
		}
		add("")
	}

	add("## Notes", "")
	return joinLines(b)
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out + "\n"
}
