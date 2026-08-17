package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/rmpato/mori/internal/entry"
)

// seeded is a small journal spread over a month, for the views that look at
// more than one day at a time.
func seeded(t *testing.T) *harness {
	t.Helper()
	aug := func(day int) entry.Date {
		return entry.Date{Year: 2026, Month: time.August, Day: day}
	}
	return newHarness(t, map[entry.Date]string{
		aug(1):  "Went to the gym. #health",
		aug(10): "Edited the Bariloche photos. #photography",
		aug(17): "Started the Go project. #go\n\nAlso the gym. #health",
	})
}

// ---------------------------------------------------------------- search ---

func TestSearchFindsDays(t *testing.T) {
	h := seeded(t)

	h.press("/")
	h.typeText("gym")

	if h.m.mode != modeSearch {
		t.Fatal("/ didn't open the search")
	}
	if len(h.m.results) != 2 {
		t.Fatalf("got %d results, want 2:\n%s", len(h.m.results), h.screen())
	}
	// The newest day is first, and it's the one the cursor starts on.
	if got := h.m.results[0].Entry.Date.Day; got != 17 {
		t.Errorf("first result is the %dth, want the 17th", got)
	}
	if !strings.Contains(h.screen(), "gym") {
		t.Errorf("the search doesn't show what it found:\n%s", h.screen())
	}
}

func TestSearchNarrowsAsYouType(t *testing.T) {
	h := seeded(t)

	h.press("/")
	h.typeText("p") // "photos", "photography", "project"
	first := len(h.m.results)
	if first != 2 {
		t.Fatalf(`"p" found %d days, want 2`, first)
	}

	h.typeText("hotos") // only the day with the photos on it
	if len(h.m.results) != 1 {
		t.Errorf(`"photos" found %d days, want 1`, len(h.m.results))
	}
}

func TestSearchOpensTheDay(t *testing.T) {
	h := seeded(t)

	h.press("/")
	h.typeText("photos")
	h.press("enter")

	if h.m.mode != modeRead {
		t.Fatal("choosing a result didn't go back to reading")
	}
	if h.m.date.Day != 10 {
		t.Errorf("opened the %dth, want the 10th", h.m.date.Day)
	}
	if !strings.Contains(h.screen(), "Bariloche") {
		t.Errorf("the page isn't on screen:\n%s", h.screen())
	}
}

func TestSearchMovesThroughResults(t *testing.T) {
	h := seeded(t)

	h.press("/")
	h.typeText("gym")
	h.press("down")
	h.press("enter")

	if h.m.date.Day != 1 {
		t.Errorf("opened the %dth, want the 1st", h.m.date.Day)
	}
}

// The selection can't be walked off either end of the list.
func TestSearchSelectionStaysInBounds(t *testing.T) {
	h := seeded(t)

	h.press("/")
	h.typeText("gym")
	for range 5 {
		h.press("down")
	}
	if h.m.selected != len(h.m.results)-1 {
		t.Errorf("selected = %d, want %d", h.m.selected, len(h.m.results)-1)
	}
	for range 5 {
		h.press("up")
	}
	if h.m.selected != 0 {
		t.Errorf("selected = %d, want 0", h.m.selected)
	}
}

func TestSearchSaysWhenItFoundNothing(t *testing.T) {
	h := seeded(t)

	h.press("/")
	h.typeText("kangaroo")

	if len(h.m.results) != 0 {
		t.Fatalf("got %d results", len(h.m.results))
	}
	if !strings.Contains(h.screen(), "nothing") {
		t.Errorf("the search says nothing about finding nothing:\n%s", h.screen())
	}
}

func TestSearchCanBeCancelled(t *testing.T) {
	h := seeded(t)

	h.press("/")
	h.typeText("gym")
	h.press("esc")

	if h.m.mode != modeRead || h.m.date != aug17 {
		t.Errorf("esc left mode %v on %v", h.m.mode, h.m.date)
	}
}

// A result from a keystroke that has since been overtaken must not land.
func TestAStaleSearchResultIsDropped(t *testing.T) {
	h := seeded(t)
	h.press("/")
	h.typeText("gym")
	good := h.m.results

	h.send(searchDoneMsg{seq: h.m.findSeq - 1, matches: nil})

	if len(h.m.results) != len(good) {
		t.Errorf("a stale result replaced %d matches with %d", len(good), len(h.m.results))
	}
}

// ------------------------------------------------------------------ tags ---

func TestTagsList(t *testing.T) {
	h := seeded(t)

	h.press("#")
	if h.m.mode != modeTags {
		t.Fatal("# didn't open the tags")
	}

	screen := h.screen()
	for _, want := range []string{"#health", "#photography", "#go", "2 days", "1 day"} {
		if !strings.Contains(screen, want) {
			t.Errorf("the tags are missing %q:\n%s", want, screen)
		}
	}
}

// Choosing a tag is choosing a search.
func TestChoosingATagSearchesForIt(t *testing.T) {
	h := seeded(t)

	h.press("#")
	h.press("enter") // health, the most used

	if h.m.mode != modeSearch {
		t.Fatalf("mode = %v, want a search", h.m.mode)
	}
	if h.m.input.Value() != "#health" {
		t.Errorf("query = %q", h.m.input.Value())
	}
	if len(h.m.results) != 2 {
		t.Errorf("got %d results, want the 2 days tagged health", len(h.m.results))
	}
}

func TestTagsOnAnUntaggedJournal(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "nothing tagged in here"})

	h.press("#")
	if !strings.Contains(h.screen(), "no tags yet") {
		t.Errorf("screen:\n%s", h.screen())
	}
}

// -------------------------------------------------------------- calendar ---

func TestCalendar(t *testing.T) {
	h := seeded(t)

	h.press("c")
	if h.m.mode != modeCalendar {
		t.Fatal("c didn't open the calendar")
	}

	screen := h.screen()
	for _, want := range []string{"August 2026", "Mo Tu We Th Fr Sa Su", "17", "a page"} {
		if !strings.Contains(screen, want) {
			t.Errorf("the calendar is missing %q:\n%s", want, screen)
		}
	}
	// It knows which days have pages.
	for _, day := range []int{1, 10, 17} {
		if !h.m.written[entry.Date{Year: 2026, Month: time.August, Day: day}] {
			t.Errorf("the %dth isn't marked as written", day)
		}
	}
	if h.m.written[entry.Date{Year: 2026, Month: time.August, Day: 2}] {
		t.Error("the 2nd is marked, but nothing was written on it")
	}
}

func TestCalendarNavigation(t *testing.T) {
	h := seeded(t)
	h.press("c")

	h.press("h")
	if h.m.cursor.Day != 16 {
		t.Errorf("h went to the %dth, want the 16th", h.m.cursor.Day)
	}
	h.press("k")
	if h.m.cursor.Day != 9 {
		t.Errorf("k went to the %dth, want a week earlier", h.m.cursor.Day)
	}
	h.press("j", "l")
	if h.m.cursor.Day != 17 {
		t.Errorf("cursor is on the %dth, want back on the 17th", h.m.cursor.Day)
	}
}

// Walking off the edge of a month loads the next one.
func TestCalendarCrossesMonths(t *testing.T) {
	h := seeded(t)
	h.press("c")

	for range 20 {
		h.press("l")
	}
	if h.m.cursor.Month != time.September {
		t.Errorf("cursor is in %v, want September", h.m.cursor.Month)
	}
	if h.m.month.Month != time.September {
		t.Errorf("the view still shows %v", h.m.month.Month)
	}
	if !strings.Contains(h.screen(), "September 2026") {
		t.Errorf("screen:\n%s", h.screen())
	}
}

func TestCalendarJumpsMonths(t *testing.T) {
	h := seeded(t)
	h.press("c")

	h.press("[")
	if h.m.month.Month != time.July {
		t.Errorf("[ went to %v, want July", h.m.month.Month)
	}
	h.press("]", "]")
	if h.m.month.Month != time.September {
		t.Errorf("] went to %v, want September", h.m.month.Month)
	}
}

// The 31st of a 31-day month has to land somewhere real in a 30-day one.
func TestCalendarJumpsToADayThatExists(t *testing.T) {
	h := newHarness(t, nil)
	h.m.cursor = entry.Date{Year: 2026, Month: time.August, Day: 31}
	h.m.mode = modeCalendar
	h.m.loadMonth()

	h.press("]") // September has 30 days
	if want := (entry.Date{Year: 2026, Month: time.September, Day: 30}); h.m.cursor != want {
		t.Errorf("cursor = %v, want %v", h.m.cursor, want)
	}
}

func TestCalendarOpensADay(t *testing.T) {
	h := seeded(t)

	h.press("c")
	h.press("k") // the 10th
	h.press("enter")

	if h.m.mode != modeRead {
		t.Fatal("choosing a day didn't go back to reading")
	}
	if h.m.date.Day != 10 {
		t.Fatalf("opened the %dth, want the 10th", h.m.date.Day)
	}
	if !strings.Contains(h.screen(), "Bariloche") {
		t.Errorf("the page isn't on screen:\n%s", h.screen())
	}
}

func TestCalendarCanBeCancelled(t *testing.T) {
	h := seeded(t)

	h.press("c")
	h.press("h", "h")
	h.press("esc")

	if h.m.mode != modeRead || h.m.date != aug17 {
		t.Errorf("esc left mode %v on %v", h.m.mode, h.m.date)
	}
}

// ------------------------------------------------------------------ mood ---

func TestMood(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "a quiet day"})

	h.press("m")
	h.typeText("calm")
	h.press("enter")

	if h.m.page.Mood != "calm" {
		t.Fatalf("mood = %q", h.m.page.Mood)
	}
	page, err := h.store.Get(aug17)
	if err != nil {
		t.Fatal(err)
	}
	if page.Mood != "calm" {
		t.Errorf("on disk the mood is %q", page.Mood)
	}
	if !strings.Contains(h.screen(), "calm") {
		t.Errorf("the mood isn't on screen:\n%s", h.screen())
	}
}

func TestMoodIsOneWord(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "a quiet day"})

	h.press("m")
	h.typeText("Quietly Hopeful")
	h.press("enter")

	if h.m.page.Mood != "quietly" {
		t.Errorf("mood = %q, want one lowercase word", h.m.page.Mood)
	}
}

func TestMoodCanBeCleared(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "a quiet day"})
	h.press("m")
	h.typeText("calm")
	h.press("enter")

	h.press("m")
	for range len("calm") {
		h.send(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	h.press("enter")

	if h.m.page.Mood != "" {
		t.Errorf("mood = %q, want it gone", h.m.page.Mood)
	}
	page, _ := h.store.Get(aug17)
	if page.Mood != "" {
		t.Errorf("on disk the mood is %q", page.Mood)
	}
}

func TestMoodCanBeCancelled(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "a quiet day"})

	h.press("m")
	h.typeText("tired")
	h.press("esc")

	if h.m.page.Mood != "" {
		t.Errorf("esc set the mood to %q", h.m.page.Mood)
	}
}

// A mood is a word about the day, and mori never says anything back about it.
func TestMoodIsNeverCommentedOn(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "a quiet day"})
	h.press("m")
	h.typeText("sad")
	h.press("enter")

	screen := strings.ToLower(h.screen())
	for _, forbidden := range []string{"trend", "average", "lately", "streak", "score"} {
		if strings.Contains(screen, forbidden) {
			t.Errorf("mori commented on the mood with %q:\n%s", forbidden, h.screen())
		}
	}
}

// ------------------------------------------------------------------ misc ---

// Every view has to survive a terminal smaller than it wants to be.
func TestEveryViewFits(t *testing.T) {
	h := seeded(t)

	for _, open := range []func(){
		func() { h.press("esc") },
		func() { h.press("c") },
		func() { h.press("esc"); h.press("/"); h.typeText("gym") },
		func() { h.press("esc"); h.press("#") },
		func() { h.press("esc"); h.press("?") },
	} {
		open()
		for _, size := range []struct{ w, h int }{{90, 24}, {40, 12}, {24, 8}, {12, 5}} {
			h.send(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			lines := strings.Split(h.screen(), "\n")
			if len(lines) > size.h {
				t.Errorf("mode %v at %dx%d: %d lines", h.m.mode, size.w, size.h, len(lines))
			}
		}
		h.send(tea.WindowSizeMsg{Width: 90, Height: 24})
	}
}
