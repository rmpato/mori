package entry

import (
	"testing"
	"time"
)

// now is a Monday, so weekday parsing has somewhere unambiguous to start.
var now = time.Date(2026, time.August, 17, 14, 30, 0, 0, time.UTC)

func date(y int, m time.Month, d int) Date { return Date{Year: y, Month: m, Day: d} }

func TestParseDate(t *testing.T) {
	tests := []struct {
		in   string
		want Date
	}{
		{"", date(2026, time.August, 17)},
		{"today", date(2026, time.August, 17)},
		{"  TODAY  ", date(2026, time.August, 17)},
		{"yesterday", date(2026, time.August, 16)},
		{"yest", date(2026, time.August, 16)},
		{"tomorrow", date(2026, time.August, 18)},

		// A weekday is the most recent one, and today when it is today.
		{"mon", date(2026, time.August, 17)},
		{"monday", date(2026, time.August, 17)},
		{"fri", date(2026, time.August, 14)},
		{"sunday", date(2026, time.August, 16)},

		// Offsets go backwards unless you ask them not to.
		{"-3d", date(2026, time.August, 14)},
		{"3d", date(2026, time.August, 14)},
		{"3 days ago", Date{}}, // not a form mori claims to speak
		{"3d ago", date(2026, time.August, 14)},
		{"+3d", date(2026, time.August, 20)},
		{"2w", date(2026, time.August, 3)},
		{"1m", date(2026, time.July, 17)},
		{"1y", date(2025, time.August, 17)},

		// Written-out dates.
		{"2026-08-17", date(2026, time.August, 17)},
		{"2020-01-05", date(2020, time.January, 5)},
		{"17 aug", date(2026, time.August, 17)},
		{"aug 17", date(2026, time.August, 17)},
		{"january 2", date(2026, time.January, 2)},
		{"2 jan 2019", date(2019, time.January, 2)},
		{"08-16", date(2026, time.August, 16)},

		// A year-less date that hasn't happened yet is last year's, because
		// mori is looking backwards.
		{"12-25", date(2025, time.December, 25)},
		{"31 dec", date(2025, time.December, 31)},
	}

	for _, tt := range tests {
		got, err := ParseDate(tt.in, now)
		if tt.want.IsZero() {
			if err == nil {
				t.Errorf("ParseDate(%q) = %v, want an error", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDate(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseDate(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseDateRejectsNonsense(t *testing.T) {
	for _, in := range []string{"someday", "next tuesday-ish", "2026-13-01", "17/08/2026x", "--"} {
		if got, err := ParseDate(in, now); err == nil {
			t.Errorf("ParseDate(%q) = %v, want an error", in, got)
		}
	}
}

func TestDateArithmetic(t *testing.T) {
	d := date(2026, time.August, 17)

	if got := d.Add(-1); got != date(2026, time.August, 16) {
		t.Errorf("Add(-1) = %v", got)
	}
	if got := d.Add(15); got != date(2026, time.September, 1) {
		t.Errorf("Add(15) = %v", got)
	}
	if got := date(2026, time.March, 1).Add(-1); got != date(2026, time.February, 28) {
		t.Errorf("crossing into February = %v", got)
	}
	if got := date(2024, time.March, 1).Add(-1); got != date(2024, time.February, 29) {
		t.Errorf("leap year = %v", got)
	}
	if got := d.Since(date(2026, time.August, 10)); got != 7 {
		t.Errorf("Since = %d, want 7", got)
	}
	if got := date(2026, time.January, 1).Since(date(2025, time.December, 31)); got != 1 {
		t.Errorf("Since across new year = %d, want 1", got)
	}
	if got := d.LastOfMonth(); got != date(2026, time.August, 31) {
		t.Errorf("LastOfMonth = %v", got)
	}
	if got := date(2026, time.February, 3).LastOfMonth(); got != date(2026, time.February, 28) {
		t.Errorf("LastOfMonth in February = %v", got)
	}
}

// Day arithmetic must not go through local midnight: a day is not always 24
// hours long, and in some zones midnight itself doesn't always exist.
func TestDateArithmeticSurvivesDST(t *testing.T) {
	for _, zone := range []string{"America/Santiago", "America/Argentina/Buenos_Aires", "Europe/Madrid"} {
		loc, err := time.LoadLocation(zone)
		if err != nil {
			t.Skipf("no zone data for %s", zone)
		}
		t.Setenv("TZ", zone)
		time.Local = loc

		// Walk a year a day at a time; every step must move exactly one day.
		d := date(2026, time.January, 1)
		for range 400 {
			next := d.Add(1)
			if next.Since(d) != 1 {
				t.Fatalf("%s: %v.Add(1) = %v, which is %d days away", zone, d, next, next.Since(d))
			}
			d = next
		}
	}
	time.Local = time.UTC
}

func TestDateFormatting(t *testing.T) {
	d := date(2026, time.August, 17)
	if got := d.String(); got != "2026-08-17" {
		t.Errorf("String = %q", got)
	}
	if got := date(2026, time.January, 5).String(); got != "2026-01-05" {
		t.Errorf("String pads = %q", got)
	}
	if got := d.Human(); got != "August 17, 2026" {
		t.Errorf("Human = %q", got)
	}
	if got := d.Full(); got != "Monday, August 17, 2026" {
		t.Errorf("Full = %q", got)
	}
}

func TestHumanRelative(t *testing.T) {
	tests := []struct {
		d    Date
		want string
	}{
		{date(2026, time.August, 17), "today"},
		{date(2026, time.August, 16), "yesterday"},
		{date(2026, time.August, 14), "friday"},
		{date(2026, time.August, 1), "1 Aug"},
		{date(2025, time.August, 1), "1 Aug 2025"},
	}
	for _, tt := range tests {
		if got := tt.d.HumanRelative(now); got != tt.want {
			t.Errorf("%v.HumanRelative = %q, want %q", tt.d, got, tt.want)
		}
	}
}
