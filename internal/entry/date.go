// Package entry holds mori's domain: a calendar day, the page you wrote on
// it, the tags in that page, and the file format both are stored as. It
// depends on nothing but the standard library, so it stays cheap to load and
// easy to test.
package entry

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Date is a calendar day: no clock, no location.
//
// A journal is indexed by the day you lived, not by an instant. Storing
// time.Time here would mean an entry written at 23:40 in Buenos Aires could
// become tomorrow's entry on a flight, which is not what anybody means by
// "what did I write on Monday".
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// DateOf is the calendar day a moment falls on, in its own location.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{Year: y, Month: m, Day: d}
}

// Today is the current calendar day, locally.
func Today() Date { return DateOf(time.Now()) }

// IsZero reports whether this is the empty Date rather than a real day.
func (d Date) IsZero() bool { return d == Date{} }

// Time is local midnight on this day, for formatting and for handing to the
// standard library.
func (d Date) Time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.Local)
}

// utc is the same day at UTC midnight. All of Date's arithmetic goes through
// this rather than through Time: local midnight is not a safe place to add
// days from, because in a zone that springs forward at midnight it doesn't
// always exist, and a day is not always 24 hours long.
func (d Date) utc() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// String is the ISO form, which is also the filename mori stores the day
// under: "2026-08-17".
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, int(d.Month), d.Day)
}

// Human is the way you'd write the date at the top of a page.
func (d Date) Human() string { return d.Time().Format("January 2, 2006") }

// Full is Human with the weekday in front, for the header of the day you're
// reading.
func (d Date) Full() string { return d.Time().Format("Monday, January 2, 2006") }

// Short is the compact form used in lists: "17 Aug".
func (d Date) Short() string { return d.Time().Format("2 Jan") }

// Weekday is which day of the week this was.
func (d Date) Weekday() time.Weekday { return d.Time().Weekday() }

// Add returns the day a number of days away. Negative goes backwards, which
// in mori is the usual direction.
func (d Date) Add(days int) Date { return DateOf(d.utc().AddDate(0, 0, days)) }

// AddMonths returns the same day-of-month a number of months away, normalised
// the way the standard library does it (31 January plus a month is 3 March).
func (d Date) AddMonths(months int) Date { return DateOf(d.utc().AddDate(0, months, 0)) }

// Before, After, and Equal compare two days chronologically.
func (d Date) Before(o Date) bool { return d.compare(o) < 0 }
func (d Date) After(o Date) bool  { return d.compare(o) > 0 }
func (d Date) Equal(o Date) bool  { return d == o }

func (d Date) compare(o Date) int {
	if d.Year != o.Year {
		return d.Year - o.Year
	}
	if d.Month != o.Month {
		return int(d.Month) - int(o.Month)
	}
	return d.Day - o.Day
}

// Since is how many days separate two dates, positive when d is later.
func (d Date) Since(o Date) int {
	return int(d.utc().Sub(o.utc()).Hours() / 24)
}

// FirstOfMonth is day 1 of this date's month.
func (d Date) FirstOfMonth() Date { return Date{Year: d.Year, Month: d.Month, Day: 1} }

// LastOfMonth is the final day of this date's month.
func (d Date) LastOfMonth() Date { return d.FirstOfMonth().AddMonths(1).Add(-1) }

// SameMonth reports whether two days fall in the same month of the same year.
func (d Date) SameMonth(o Date) bool { return d.Year == o.Year && d.Month == o.Month }

var weekdays = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "weds": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

// offsetRe matches "-3d", "3d", "+2w", "1m", and the "ago" spellings.
var offsetRe = regexp.MustCompile(`^([-+]?)(\d+)\s*([dwmy])(\s+ago)?$`)

// dateLayouts are the written-out forms mori accepts, tried in order.
var dateLayouts = []string{
	"2006-01-02",
	"2006/01/02",
	"02 Jan 2006",
	"2 Jan 2006",
	"Jan 2 2006",
	"Jan 2, 2006",
	"January 2 2006",
	"January 2, 2006",
}

// partialLayouts are the same forms with the year left off.
var partialLayouts = []string{
	"01-02",
	"01/02",
	"02 Jan",
	"2 Jan",
	"Jan 2",
	"January 2",
}

// ParseDate turns a friendly string into a calendar day, relative to now.
//
// It speaks tuki's date vocabulary, with the sign flipped: tuki looks forward
// and mori looks back, so a bare offset like "3d" means three days ago and a
// weekday name means the most recent one, not the next one.
//
//	today, tod, now          yesterday, yest
//	fri, friday              -3d, 3d, 3 days ago, -2w, 1m, 1y
//	2026-08-17               08-17, 17 Aug, Jan 2, January 2, 2026
func ParseDate(s string, now time.Time) (Date, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ") // squeeze inner whitespace
	today := DateOf(now)

	switch s {
	case "", "today", "tod", "now":
		return today, nil
	case "yesterday", "yest", "yd":
		return today.Add(-1), nil
	case "tomorrow", "tom", "tmr":
		return today.Add(1), nil
	case "last week":
		return today.Add(-7), nil
	case "last month":
		return today.AddMonths(-1), nil
	}

	// A weekday name means the most recent one, and "fri" on a Friday is today.
	if wd, ok := weekdays[s]; ok {
		back := (int(today.Weekday()) - int(wd) + 7) % 7
		return today.Add(-back), nil
	}

	if m := offsetRe.FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			return Date{}, fmt.Errorf("mori can't read the offset in %q", s)
		}
		// Backwards unless you explicitly ask to go forward.
		if m[1] != "+" {
			n = -n
		}
		switch m[3] {
		case "d":
			return today.Add(n), nil
		case "w":
			return today.Add(7 * n), nil
		case "m":
			return today.AddMonths(n), nil
		case "y":
			return today.AddMonths(12 * n), nil
		}
	}

	// Titlecase so "17 aug" and "january 2" reach time.Parse's month names.
	cased := titleWords(s)
	for _, layout := range dateLayouts {
		// Parsed in UTC because only the calendar fields are wanted; a
		// date-only layout has no business resolving a local instant.
		if t, err := time.Parse(layout, cased); err == nil {
			return DateOf(t), nil
		}
	}

	// A date without a year means this year, or last year if that hasn't
	// happened yet — mori is looking backwards, so "12-25" in March is the
	// Christmas you remember, not the one still coming.
	for _, layout := range partialLayouts {
		t, err := time.Parse(layout, cased)
		if err != nil {
			continue
		}
		d := Date{Year: today.Year, Month: t.Month(), Day: t.Day()}
		if d.After(today) {
			d.Year--
		}
		return d, nil
	}

	return Date{}, fmt.Errorf("mori doesn't know when %q is", s)
}

// titleWords uppercases the first letter of each word, which is all
// time.Parse needs to recognise a month name.
func titleWords(s string) string {
	words := strings.Split(s, " ")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// Human renders a date the way you'd say it out loud, given where today is.
func (d Date) HumanRelative(now time.Time) string {
	today := DateOf(now)
	days := d.Since(today)

	switch {
	case days == 0:
		return "today"
	case days == -1:
		return "yesterday"
	case days == 1:
		return "tomorrow"
	case days < 0 && days > -7:
		return strings.ToLower(d.Weekday().String())
	}
	if d.Year != today.Year {
		return d.Time().Format("2 Jan 2006")
	}
	return d.Short()
}
