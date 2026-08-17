package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rmpato/mori/internal/entry"
)

func seedJournal(t *testing.T) (func(args ...string) (string, error), entry.Date) {
	t.Helper()
	s, run := journal(t)

	aug := func(day int) entry.Date {
		return entry.Date{Year: 2026, Month: time.August, Day: day}
	}
	write(t, s, aug(1), entry.Entry{Body: "Went to the gym. #health"})
	write(t, s, aug(10), entry.Entry{Mood: "calm", Body: "Edited the Bariloche photos. #photography"})
	write(t, s, aug(17), entry.Entry{Body: "Started the Go project. #go\n\nAlso the gym. #health"})

	return run, aug(17)
}

func TestSearch(t *testing.T) {
	run, _ := seedJournal(t)

	tests := []struct {
		query string
		want  []string
		miss  []string
	}{
		{query: "gym", want: []string{"2026-08-01", "2026-08-17"}, miss: []string{"2026-08-10"}},
		{query: "photo", want: []string{"2026-08-10"}, miss: []string{"2026-08-01"}},
		{query: "#photography", want: []string{"2026-08-10"}},
		{query: "mood:calm", want: []string{"2026-08-10"}},
		{query: "gym since:2026-08-05", want: []string{"2026-08-17"}, miss: []string{"2026-08-01"}},
		{query: "nothing at all matches this", miss: []string{"2026-08-01", "2026-08-10", "2026-08-17"}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			out, err := run(append([]string{"search"}, strings.Fields(tt.query)...)...)
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("missing %s:\n%s", want, out)
				}
			}
			for _, miss := range tt.miss {
				if strings.Contains(out, miss) {
					t.Errorf("unexpectedly found %s:\n%s", miss, out)
				}
			}
		})
	}
}

// The newest day comes first: looking back starts from where you are.
func TestSearchGoesBackwards(t *testing.T) {
	run, _ := seedJournal(t)

	out, err := run("search", "gym")
	if err != nil {
		t.Fatal(err)
	}
	first, second := strings.Index(out, "2026-08-17"), strings.Index(out, "2026-08-01")
	if first < 0 || second < 0 || first > second {
		t.Errorf("wrong order:\n%s", out)
	}
}

func TestSearchQuotedPhrase(t *testing.T) {
	run, _ := seedJournal(t)

	out, err := run("search", "the Go project")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "2026-08-17") {
		t.Errorf("the words weren't found:\n%s", out)
	}

	// The same words in an order the page doesn't use.
	out, err = run("search", "project the Go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "2026-08-17") {
		t.Errorf("word order shouldn't matter:\n%s", out)
	}
}

func TestSearchJSON(t *testing.T) {
	run, _ := seedJournal(t)

	out, err := run("search", "gym", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got []struct {
		Date    string   `json:"date"`
		Excerpt string   `json:"excerpt"`
		Tags    []string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2", len(got))
	}
	if got[0].Date != "2026-08-17" || !strings.Contains(got[0].Excerpt, "gym") {
		t.Errorf("first match = %+v", got[0])
	}
}

func TestSearchLimit(t *testing.T) {
	run, _ := seedJournal(t)

	out, err := run("search", "gym", "-n", "1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "2026-08-01") {
		t.Errorf("--limit didn't stop early:\n%s", out)
	}
}

func TestSearchRejectsADateItCannotRead(t *testing.T) {
	run, _ := seedJournal(t)
	if _, err := run("search", "gym", "since:someday"); err == nil {
		t.Error("search accepted a date it can't resolve")
	}
}

// Down a pipe, a search that found nothing says nothing.
func TestSearchWithNoMatchesIsSilentDownAPipe(t *testing.T) {
	run, _ := seedJournal(t)
	out, err := run("search", "kangaroo")
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("search printed %q", out)
	}
}

func TestTags(t *testing.T) {
	run, _ := seedJournal(t)

	out, err := run("tags")
	if err != nil {
		t.Fatal(err)
	}
	// health is on two days, so it leads; the rest are alphabetical.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d tags:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "health\t2") {
		t.Errorf("first line = %q, want health with two days", lines[0])
	}
}

func TestTagsJSON(t *testing.T) {
	run, _ := seedJournal(t)

	out, err := run("tags", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got []struct {
		Tag  string `json:"tag"`
		Days int    `json:"days"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(got) != 3 || got[0].Tag != "health" || got[0].Days != 2 {
		t.Errorf("tags = %+v", got)
	}
}

func TestTagsOnAnUntaggedJournal(t *testing.T) {
	s, run := journal(t)
	write(t, s, entry.Today(), entry.Entry{Body: "no tags in here"})

	out, err := run("tags")
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("tags printed %q", out)
	}
}

func TestList(t *testing.T) {
	run, _ := seedJournal(t)

	out, err := run("list")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2026-08-01", "2026-08-10", "2026-08-17"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}

	out, err = run("list", "--since", "2026-08-05")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "2026-08-01") {
		t.Errorf("--since didn't bound the list:\n%s", out)
	}
}
