// Package ui holds everything about how mori looks and the few things mori
// says. tuki is amber and mischievous; mori is moss and patient. Same bones,
// different temperature.
package ui

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme is a resolved palette and the styles built from it. Build one with
// New and pass it around; it is cheap to copy.
type Theme struct {
	IsDark bool

	// Palette.
	Brand  color.Color // mori's own green
	Text   color.Color // the writing itself
	Muted  color.Color // dates, counts, secondary things
	Faint  color.Color // rules and barely-there details
	Accent color.Color // autumn gold, used sparingly

	// Styles.
	Logo    lipgloss.Style
	Date    lipgloss.Style
	Weekday lipgloss.Style
	Rule    lipgloss.Style
	Body    lipgloss.Style
	Time    lipgloss.Style // the "23:04" of a section heading
	Tag     lipgloss.Style
	Mood    lipgloss.Style
	Hint    lipgloss.Style
	Aside   lipgloss.Style
	Prompt  lipgloss.Style
	Warn    lipgloss.Style
}

// New builds the theme for a light or dark terminal.
func New(isDark bool) Theme {
	c := lipgloss.LightDark(isDark)

	t := Theme{
		IsDark: isDark,

		Brand:  c(lipgloss.Color("#3E7A52"), lipgloss.Color("#86C79A")),
		Text:   c(lipgloss.Color("#1F2328"), lipgloss.Color("#DCDEE0")),
		Muted:  c(lipgloss.Color("#6B7075"), lipgloss.Color("#9198A0")),
		Faint:  c(lipgloss.Color("#B5BAC0"), lipgloss.Color("#5A6067")),
		Accent: c(lipgloss.Color("#8A6A00"), lipgloss.Color("#DCC96B")),
	}

	t.Logo = lipgloss.NewStyle().Foreground(t.Brand).Bold(true)
	t.Date = lipgloss.NewStyle().Foreground(t.Muted)
	t.Weekday = lipgloss.NewStyle().Foreground(t.Brand)
	t.Rule = lipgloss.NewStyle().Foreground(t.Faint)
	t.Body = lipgloss.NewStyle().Foreground(t.Text)
	t.Time = lipgloss.NewStyle().Foreground(t.Faint)
	t.Tag = lipgloss.NewStyle().Foreground(t.Brand)
	t.Mood = lipgloss.NewStyle().Foreground(t.Accent)
	t.Hint = lipgloss.NewStyle().Foreground(t.Faint)
	t.Aside = lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	t.Prompt = lipgloss.NewStyle().Foreground(t.Brand)
	t.Warn = lipgloss.NewStyle().Foreground(t.Accent)

	return t
}

// Rule draws the thin line mori puts above and below a page.
func (t Theme) Line(width int) string {
	if width < 1 {
		width = 1
	}
	return t.Rule.Render(strings.Repeat("─", width))
}

// PrefersDark guesses whether the terminal has a dark background without
// asking it, which keeps `mori today` instant. MORI_THEME wins; then the
// COLORFGBG convention many terminals set; otherwise dark, because most
// terminals are.
//
// The full-screen interface doesn't use this: it asks the terminal properly
// and asynchronously, through Bubble Tea.
func PrefersDark() bool {
	switch strings.ToLower(os.Getenv("MORI_THEME")) {
	case "light":
		return false
	case "dark":
		return true
	}
	// COLORFGBG looks like "15;0" — foreground;background, as ANSI indices.
	if v := os.Getenv("COLORFGBG"); v != "" {
		parts := strings.Split(v, ";")
		switch parts[len(parts)-1] {
		case "7", "15":
			return false
		case "0", "8":
			return true
		}
	}
	return true
}
