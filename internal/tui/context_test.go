package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/facts"
)

// fakeSource stands in for tuki, so the pane can be tested without one
// installed.
type fakeSource struct {
	snap facts.Snapshot
	err  error
}

func (f fakeSource) Day(entry.Date) (facts.Snapshot, error) { return f.snap, f.err }

func withTuki(t *testing.T, seed map[entry.Date]string) *harness {
	t.Helper()
	return newHarnessWith(t, seed, fakeSource{snap: facts.Snapshot{
		Source: "tuki",
		Done:   []facts.Item{{Text: "Finish Go project", Tag: "work"}, {Text: "Go to gym", Tag: "home"}},
		Todo:   []facts.Item{{Text: "Edit photos", Tag: "photography"}},
	}})
}

func TestContextShowsWhatTukiKnows(t *testing.T) {
	h := withTuki(t, nil)

	h.press("tab")
	if h.m.mode != modeContext {
		t.Fatal("tab didn't open the context")
	}

	screen := h.screen()
	for _, want := range []string{"Finish Go project", "Go to gym", "Edit photos", "#work", "#photography"} {
		if !strings.Contains(screen, want) {
			t.Errorf("the context is missing %q:\n%s", want, screen)
		}
	}
}

// mori works perfectly well with no tuki, and says nothing about it.
func TestContextDoesNothingWithoutTuki(t *testing.T) {
	h := newHarness(t, nil) // no source at all

	h.press("tab")
	if h.m.mode != modeRead {
		t.Errorf("tab opened a context that has no source: mode = %v", h.m.mode)
	}
	if strings.Contains(strings.ToLower(h.screen()), "tuki") {
		t.Errorf("mori mentioned tuki when it isn't there:\n%s", h.screen())
	}
}

func TestContextWithNothingToSay(t *testing.T) {
	h := newHarnessWith(t, nil, fakeSource{snap: facts.Snapshot{Source: "tuki"}})

	h.press("tab")
	if !strings.Contains(h.screen(), "nothing for this day") {
		t.Errorf("screen:\n%s", h.screen())
	}
}

// A tuki that can't be read is a passing note, not a broken journal.
func TestContextSurvivesABrokenTuki(t *testing.T) {
	h := newHarnessWith(t, map[entry.Date]string{aug17: "a quiet day"},
		fakeSource{err: errors.New("tasks.json is unreadable")})

	h.press("tab")
	if !strings.Contains(h.screen(), "unreadable") {
		t.Errorf("mori said nothing about the problem:\n%s", h.screen())
	}
	h.press("esc")
	if !strings.Contains(h.screen(), "a quiet day") {
		t.Errorf("the journal itself was disturbed:\n%s", h.screen())
	}
}

// The facts become a scaffold to write under — headings and your own task
// text, and nothing that pretends to be your writing.
func TestStartingAPageFromTheContext(t *testing.T) {
	h := withTuki(t, nil)

	h.press("tab")
	h.press("enter")

	if h.m.mode != modeWrite {
		t.Fatalf("mode = %v, want the cursor in the page", h.m.mode)
	}
	body := h.m.ta.Value()
	for _, want := range []string{"## Things I did", "- Finish Go project", "## Things I didn't get to", "- Edit photos", "## Notes"} {
		if !strings.Contains(body, want) {
			t.Errorf("the page is missing %q:\n%s", want, body)
		}
	}

	h.press("esc")
	if !strings.Contains(h.onDisk(aug17), "## Things I did") {
		t.Errorf("on disk = %q", h.onDisk(aug17))
	}
}

// A starting point is only a starting point when there is nothing there yet.
func TestTheContextNeverOverwritesWriting(t *testing.T) {
	h := withTuki(t, map[entry.Date]string{aug17: "I already wrote this."})

	h.press("tab")
	h.press("enter")

	if got := h.onDisk(aug17); got != "I already wrote this." {
		t.Errorf("on disk = %q, want the writing untouched", got)
	}
	if !strings.Contains(h.screen(), "already writing") {
		t.Errorf("mori didn't say why it declined:\n%s", h.screen())
	}
}

// The context is read-only in the strongest sense: nothing mori does here can
// reach back into the task list.
func TestTheContextIsOnlyEverRead(t *testing.T) {
	src := &countingSource{}
	h := newHarnessWith(t, nil, src)

	h.press("tab")
	h.press("enter")
	h.press("esc")
	h.press("tab")

	if src.reads == 0 {
		t.Error("the context was never read")
	}
	// facts.Source has no way to write, which is the point: there is no
	// method here that could change a task even by mistake.
	var _ interface {
		Day(entry.Date) (facts.Snapshot, error)
	} = src
}

type countingSource struct{ reads int }

func (c *countingSource) Day(entry.Date) (facts.Snapshot, error) {
	c.reads++
	return facts.Snapshot{Source: "tuki", Done: []facts.Item{{Text: "something"}}}, nil
}

func TestContextCanBeCancelled(t *testing.T) {
	h := withTuki(t, map[entry.Date]string{aug17: "a quiet day"})

	h.press("tab")
	h.press("esc")

	if h.m.mode != modeRead {
		t.Errorf("mode = %v", h.m.mode)
	}
	if got := h.onDisk(aug17); got != "a quiet day" {
		t.Errorf("on disk = %q", got)
	}
}
