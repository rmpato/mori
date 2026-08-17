package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rmpato/mori/internal/entry"
)

func TestParseMonth(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	aug := entry.Date{Year: 2026, Month: time.August, Day: 1}

	tests := map[string]entry.Date{
		"":           aug,
		"august":     aug,
		"aug":        aug,
		"AUGUST":     aug,
		"2026-08":    aug,
		"2026-08-17": aug,
		"last month": {Year: 2026, Month: time.July, Day: 1},
		"july":       {Year: 2026, Month: time.July, Day: 1},
		// A month that hasn't come round yet this year is last year's.
		"december": {Year: 2025, Month: time.December, Day: 1},
		"-2m":      {Year: 2026, Month: time.June, Day: 1},
	}

	for in, want := range tests {
		got, err := parseMonth(in, now)
		if err != nil {
			t.Errorf("parseMonth(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseMonth(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseMonthRejectsNonsense(t *testing.T) {
	now := time.Now()
	for _, in := range []string{"someday", "the month of sundays"} {
		if got, err := parseMonth(in, now); err == nil {
			t.Errorf("parseMonth(%q) = %v, want an error", in, got)
		}
	}
}

func TestLookingBack(t *testing.T) {
	s, run := journal(t)
	aug := func(day int) entry.Date {
		return entry.Date{Year: 2026, Month: time.August, Day: day}
	}
	write(t, s, aug(2), entry.Entry{Body: "Walked to the lake. #photography"})
	write(t, s, aug(10), entry.Entry{Body: "Edited photos. #photography"})
	write(t, s, aug(17), entry.Entry{Body: "Started the Go project. #go"})
	// A day in another month, which must not be counted.
	write(t, s, entry.Date{Year: 2026, Month: time.July, Day: 30}, entry.Entry{Body: "July. #go"})

	out, err := run("looking-back", "2026-08", "--json")
	if err != nil {
		t.Fatalf("looking-back: %v", err)
	}

	var got overview
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if got.Month != "August 2026" {
		t.Errorf("Month = %q", got.Month)
	}
	if got.Days != 3 {
		t.Errorf("Days = %d, want 3", got.Days)
	}
	if got.First != "2026-08-02" || got.Last != "2026-08-17" {
		t.Errorf("span = %s..%s", got.First, got.Last)
	}
	if len(got.Tags) != 2 || got.Tags[0].Tag != "photography" || got.Tags[0].Days != 2 {
		t.Errorf("Tags = %+v", got.Tags)
	}
	if got.Tuki != nil {
		t.Errorf("Tuki = %+v, want nothing without tuki", got.Tuki)
	}
}

func TestLookingBackAtAnEmptyMonth(t *testing.T) {
	_, run := journal(t)

	out, err := run("looking-back", "2020-01")
	if err != nil {
		t.Fatalf("looking-back: %v", err)
	}
	// Nothing down a pipe; a gentle line at a terminal. Either way, no
	// remark about the emptiness beyond the fact of it.
	if strings.Contains(strings.ToLower(out), "should") {
		t.Errorf("mori told the user off:\n%s", out)
	}
}

// The one thing this must never become is a report card.
func TestLookingBackKeepsNoScore(t *testing.T) {
	s, run := journal(t)
	for day := 1; day <= 5; day++ {
		write(t, s, entry.Date{Year: 2026, Month: time.August, Day: day},
			entry.Entry{Body: "a day. #go"})
	}

	out, err := run("looking-back", "2026-08")
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(out)
	for _, forbidden := range []string{
		"%", "streak", "average", "productiv", "score", "goal", "target",
		"more than", "less than", "last month", "keep it up", "well done",
	} {
		if strings.Contains(lower, forbidden) {
			t.Errorf("looking-back said %q:\n%s", forbidden, out)
		}
	}
}

func TestLookingBackWithTuki(t *testing.T) {
	_, run := journal(t)

	// A month of tuki activity, and a journal day inside it.
	tasks := `{"version": 1, "tasks": [
		{"text": "Finish Go project", "tag": "work", "done": true,
		 "created_at": "2026-08-01T09:00:00Z", "done_at": "2026-08-17T18:00:00Z"},
		{"text": "Select photos", "tag": "photography", "done": true,
		 "created_at": "2026-08-01T09:00:00Z", "done_at": "2026-08-10T18:00:00Z"},
		{"text": "Edit photos", "tag": "photography", "done": true,
		 "created_at": "2026-08-01T09:00:00Z", "done_at": "2026-08-11T18:00:00Z"},
		{"text": "Something in July", "tag": "work", "done": true,
		 "created_at": "2026-07-01T09:00:00Z", "done_at": "2026-07-05T18:00:00Z"}
	]}`
	t.Setenv("TUKI_FILE", writeTemp(t, "tasks.json", tasks))

	out, err := run("looking-back", "2026-08", "--json")
	if err != nil {
		t.Fatalf("looking-back: %v", err)
	}
	var got overview
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}

	if got.Tuki == nil {
		t.Fatal("nothing from tuki")
	}
	// Three finished in August; the July one belongs to July.
	if got.Tuki.Finished != 3 {
		t.Errorf("Finished = %d, want 3", got.Tuki.Finished)
	}
	if len(got.Tuki.Tags) == 0 || got.Tuki.Tags[0].Tag != "photography" || got.Tuki.Tags[0].Finished != 2 {
		t.Errorf("Tags = %+v", got.Tuki.Tags)
	}
}
