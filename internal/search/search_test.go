package search

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/store"
)

var now = time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)

func date(y int, m time.Month, d int) entry.Date {
	return entry.Date{Year: y, Month: m, Day: d}
}

func journal(t *testing.T, pages map[entry.Date]entry.Entry) *store.Store {
	t.Helper()
	s := store.New(filepath.Join(t.TempDir(), "journal"))
	for d, e := range pages {
		e.Date = d
		if err := s.Put(e); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func parse(t *testing.T, s string) Query {
	t.Helper()
	q, err := Parse(s, now)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return q
}

func TestParse(t *testing.T) {
	q := parse(t, `photography "the zine idea" #go mood:calm since:2026-01-01 until:yesterday`)

	if want := []string{"photography"}; !reflect.DeepEqual(q.Terms, want) {
		t.Errorf("Terms = %v, want %v", q.Terms, want)
	}
	if want := []string{"the zine idea"}; !reflect.DeepEqual(q.Phrases, want) {
		t.Errorf("Phrases = %v, want %v", q.Phrases, want)
	}
	if want := []string{"go"}; !reflect.DeepEqual(q.Tags, want) {
		t.Errorf("Tags = %v, want %v", q.Tags, want)
	}
	if q.Mood != "calm" {
		t.Errorf("Mood = %q", q.Mood)
	}
	if want := date(2026, time.January, 1); q.Since != want {
		t.Errorf("Since = %v, want %v", q.Since, want)
	}
	if want := date(2026, time.August, 16); q.Until != want {
		t.Errorf("Until = %v, want %v", q.Until, want)
	}
}

func TestParseIsForgiving(t *testing.T) {
	// A quote you haven't closed yet is a phrase in progress, not a mistake.
	q := parse(t, `"half a phrase`)
	if want := []string{"half a phrase"}; !reflect.DeepEqual(q.Phrases, want) {
		t.Errorf("Phrases = %v, want %v", q.Phrases, want)
	}

	// Terms are lowercased so the search doesn't care how you typed it.
	if q := parse(t, "Photography GO"); !reflect.DeepEqual(q.Terms, []string{"photography", "go"}) {
		t.Errorf("Terms = %v", q.Terms)
	}

	// A colon in ordinary writing is not a field.
	if q := parse(t, "note:to self"); !reflect.DeepEqual(q.Terms, []string{"note:to", "self"}) {
		t.Errorf("Terms = %v", q.Terms)
	}

	if q := parse(t, "   "); !q.IsEmpty() {
		t.Error("an empty search isn't empty")
	}
}

func TestParseRejectsADateItCannotRead(t *testing.T) {
	if _, err := Parse("since:someday", now); err == nil {
		t.Error("Parse accepted a date it can't resolve")
	}
}

func TestMatchTerms(t *testing.T) {
	page := entry.Entry{Body: "I finally started the Go project. Went to the gym."}

	tests := []struct {
		query string
		want  bool
	}{
		{"go", true},
		{"GO", true},
		{"project", true},
		{"go project", true},         // both words, in any order
		{"project go", true},         //
		{"go photography", false},    // one of them is missing
		{"proj", true},               // the start of a word
		{"roject", false},            // the middle of one is not a match
		{"gym", true},                //
		{`"the Go project"`, true},   // an exact phrase
		{`"the go project."`, true},  // still exact, still case-insensitive
		{`"started the gym"`, false}, // not what the page says
	}

	for _, tt := range tests {
		_, ok := parse(t, tt.query).Match(page)
		if ok != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.query, ok, tt.want)
		}
	}
}

func TestMatchTagsAndMood(t *testing.T) {
	page := entry.Entry{Mood: "calm", Body: "edited the #photography photos #go"}

	for _, q := range []string{"#photography", "tag:photography", "#photography #go", "mood:calm", "#go photos"} {
		if _, ok := parse(t, q).Match(page); !ok {
			t.Errorf("Match(%q) = false", q)
		}
	}
	for _, q := range []string{"#travel", "mood:tired", "#photography #travel"} {
		if _, ok := parse(t, q).Match(page); ok {
			t.Errorf("Match(%q) = true", q)
		}
	}
}

func TestExcerptPointsAtTheHit(t *testing.T) {
	page := entry.Entry{Body: "Slow morning.\n\n## 23:04\n\nWent back to the Go project after dinner."}

	m, ok := parse(t, "project").Match(page)
	if !ok {
		t.Fatal("no match")
	}
	if want := "Went back to the Go project after dinner."; m.Excerpt != want {
		t.Errorf("Excerpt = %q, want %q", m.Excerpt, want)
	}
	if m.Hit != "project" {
		t.Errorf("Hit = %q", m.Hit)
	}
}

// The hit comes back as it was written, so it can be found again in the line
// however it was capitalised.
func TestHitKeepsItsOriginalSpelling(t *testing.T) {
	page := entry.Entry{Body: "Bariloche was beautiful."}
	m, ok := parse(t, "bariloche").Match(page)
	if !ok {
		t.Fatal("no match")
	}
	if m.Hit != "Bariloche" {
		t.Errorf("Hit = %q, want %q", m.Hit, "Bariloche")
	}
}

// A section heading is scaffolding, not writing, so it never becomes the
// excerpt for a match found further down.
func TestExcerptSkipsSectionHeadings(t *testing.T) {
	page := entry.Entry{Body: "## 09:14\n\nthe gym was busy"}
	m, _ := parse(t, "gym").Match(page)
	if m.Excerpt != "the gym was busy" {
		t.Errorf("Excerpt = %q", m.Excerpt)
	}
}

func TestRunGoesBackwards(t *testing.T) {
	s := journal(t, map[entry.Date]entry.Entry{
		date(2026, time.August, 1):  {Body: "the gym"},
		date(2026, time.August, 10): {Body: "no mention"},
		date(2026, time.August, 17): {Body: "the gym again"},
	})

	got, err := All(s, parse(t, "gym"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2", len(got))
	}
	if got[0].Entry.Date != date(2026, time.August, 17) {
		t.Errorf("first match is %v, want the most recent day", got[0].Entry.Date)
	}
}

func TestRunRespectsDateBounds(t *testing.T) {
	s := journal(t, map[entry.Date]entry.Entry{
		date(2025, time.December, 31): {Body: "the gym"},
		date(2026, time.August, 17):   {Body: "the gym"},
	})

	got, err := All(s, parse(t, "gym since:2026-01-01"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Entry.Date.Year != 2026 {
		t.Errorf("got %v, want only the 2026 day", got)
	}
}

func TestLimitStopsEarly(t *testing.T) {
	pages := map[entry.Date]entry.Entry{}
	for day := 1; day <= 10; day++ {
		pages[date(2026, time.August, day)] = entry.Entry{Body: "the gym"}
	}
	s := journal(t, pages)

	got, err := All(s, parse(t, "gym"), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d matches, want 3", len(got))
	}
}

// An empty search is every day you wrote, which is what makes it a listing.
func TestAnEmptySearchIsEveryDay(t *testing.T) {
	s := journal(t, map[entry.Date]entry.Entry{
		date(2026, time.August, 1):  {Body: "one"},
		date(2026, time.August, 17): {Body: "two"},
	})

	got, err := All(s, Query{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("got %d days, want 2", len(got))
	}
}

func TestTagCounts(t *testing.T) {
	s := journal(t, map[entry.Date]entry.Entry{
		date(2026, time.August, 1):  {Body: "#photography #go"},
		date(2026, time.August, 2):  {Body: "#photography"},
		date(2026, time.August, 3):  {Body: "#photography and #photography again"},
		date(2026, time.August, 17): {Body: "nothing tagged here"},
	})

	got, err := Tags(s, entry.Date{}, entry.Date{})
	if err != nil {
		t.Fatal(err)
	}
	want := []TagCount{{Tag: "photography", Days: 3}, {Tag: "go", Days: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tags = %v, want %v", got, want)
	}
}

func TestSearchingAnEmptyJournal(t *testing.T) {
	s := store.New(filepath.Join(t.TempDir(), "journal"))
	got, err := All(s, parse(t, "anything"), 0)
	if err != nil {
		t.Fatalf("searching an empty journal: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}
