package ui

import (
	"os"
	"strings"
	"time"

	"github.com/rmpato/mori/internal/entry"
)

// Season is which quarter of the year a day falls in. mori uses it for one
// small thing — the glyph beside the date — so that scrolling back through a
// year quietly changes weather.
type Season int

const (
	Spring Season = iota
	Summer
	Autumn
	Winter
)

// Glyph is the season's mark.
func (s Season) Glyph() string {
	switch s {
	case Spring:
		return "🌱"
	case Summer:
		return "🌿"
	case Autumn:
		return "🍂"
	default:
		return "❄"
	}
}

// Name is the season, lowercase.
func (s Season) Name() string {
	switch s {
	case Spring:
		return "spring"
	case Summer:
		return "summer"
	case Autumn:
		return "autumn"
	default:
		return "winter"
	}
}

// SeasonOf is the season a day falls in, by meteorological month rather than
// by solstice — close enough for a glyph, and it doesn't need a table.
//
// South flips the calendar, because a journal written in Bariloche should not
// draw a snowflake in July.
func SeasonOf(d entry.Date, south bool) Season {
	var s Season
	switch d.Month {
	case time.March, time.April, time.May:
		s = Spring
	case time.June, time.July, time.August:
		s = Summer
	case time.September, time.October, time.November:
		s = Autumn
	default:
		s = Winter
	}
	if south {
		s = (s + 2) % 4
	}
	return s
}

// SouthernHemisphere reads MORI_HEMISPHERE. It only decides which glyph goes
// next to the date, so guessing wrong costs nothing but a wrong leaf.
func SouthernHemisphere() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MORI_HEMISPHERE"))) {
	case "south", "southern", "s":
		return true
	default:
		return false
	}
}
