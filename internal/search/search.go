// Package search answers questions about a journal: which days mention a
// word, which carry a tag, how often a tag has been used.
//
// It is a plain scan, newest day first, and it streams — so the first results
// appear immediately and nothing has to hold a decade of pages in memory. Ten
// years of daily writing is a few thousand small files, which is nothing to a
// filesystem. If that ever stops being true, an indexed store slots in behind
// store.Journal without any of this changing.
package search

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/store"
)

// ErrStop ends a search early without it counting as a failure.
var ErrStop = store.ErrStop

// Query is a parsed search. An empty Query matches every day, which is what
// makes it useful for listing as well as for searching.
type Query struct {
	Terms   []string // whole words, or the starts of words
	Phrases []string // quoted: matched literally, anywhere
	Tags    []string
	Mood    string
	Since   entry.Date // zero means as far back as there is
	Until   entry.Date // zero means up to the last day written
}

// Match is one day that answered.
type Match struct {
	Entry entry.Entry

	// Excerpt is the line the first hit fell on, and Hit is the text that
	// matched, exactly as it was written, so it can be picked out again.
	Excerpt string
	Hit     string
}

// IsEmpty reports whether the query asks anything about the writing itself,
// as opposed to only bounding the dates.
func (q Query) IsEmpty() bool {
	return len(q.Terms) == 0 && len(q.Phrases) == 0 && len(q.Tags) == 0 && q.Mood == ""
}

// Parse reads a search the way you'd type it.
//
//	photography              a word, or the start of one
//	"the zine idea"          an exact phrase
//	#go                      a tag
//	mood:calm                a mood
//	since:2026-01-01         bounds; until: and before: work too
func Parse(s string, now time.Time) (Query, error) {
	var q Query

	for _, tok := range tokenize(s) {
		if tok.text == "" {
			continue
		}
		if tok.quoted {
			q.Phrases = append(q.Phrases, tok.text)
			continue
		}

		key, value, isPair := strings.Cut(tok.text, ":")
		if isPair && value != "" {
			switch strings.ToLower(key) {
			case "since", "from", "after":
				d, err := entry.ParseDate(value, now)
				if err != nil {
					return Query{}, err
				}
				q.Since = d
				continue
			case "until", "to", "before":
				d, err := entry.ParseDate(value, now)
				if err != nil {
					return Query{}, err
				}
				q.Until = d
				continue
			case "tag":
				if t := entry.NormalizeTag(value); t != "" {
					q.Tags = append(q.Tags, t)
				}
				continue
			case "mood":
				q.Mood = entry.NormalizeMood(value)
				continue
			}
		}

		if strings.HasPrefix(tok.text, "#") {
			if t := entry.NormalizeTag(tok.text); t != "" {
				q.Tags = append(q.Tags, t)
			}
			continue
		}

		q.Terms = append(q.Terms, strings.ToLower(tok.text))
	}

	return q, nil
}

// Match tests one day against the query.
//
// Every term and phrase has to appear: searching two words means you're
// looking for the day that had both. Bare terms match the start of a word, so
// "photo" finds "photography"; a quoted phrase is matched literally, which is
// how you ask for exactly what you typed.
func (q Query) Match(e entry.Entry) (Match, bool) {
	if q.Mood != "" && e.Mood != q.Mood {
		return Match{}, false
	}
	for _, tag := range q.Tags {
		if !e.HasTag(tag) {
			return Match{}, false
		}
	}
	if len(q.Terms) == 0 && len(q.Phrases) == 0 {
		// Nothing to look for in the prose: the day has already qualified on
		// its tags and mood, so its opening line is the thing to show.
		return Match{Entry: e, Excerpt: e.Excerpt(0)}, true
	}

	body := strings.ToLower(e.Body)
	for _, t := range q.Terms {
		if indexWord(body, t) < 0 {
			return Match{}, false
		}
	}
	for _, p := range q.Phrases {
		if !strings.Contains(body, strings.ToLower(p)) {
			return Match{}, false
		}
	}

	m := Match{Entry: e}
	m.Excerpt, m.Hit = q.excerpt(e)
	return m, true
}

// excerpt finds the line the first hit fell on. It works line by line rather
// than by offset into the whole page, so nothing has to map positions between
// the original text and its lowercased copy.
func (q Query) excerpt(e entry.Entry) (excerpt, hit string) {
	for raw := range strings.Lines(e.Body) {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r\n"))
		if line == "" || isHeading(line) {
			continue
		}
		lower := strings.ToLower(line)

		best, length := -1, 0
		for _, t := range q.Terms {
			if i := indexWord(lower, t); i >= 0 && (best < 0 || i < best) {
				best, length = i, len(t)
			}
		}
		for _, p := range q.Phrases {
			if i := strings.Index(lower, strings.ToLower(p)); i >= 0 && (best < 0 || i < best) {
				best, length = i, len(p)
			}
		}
		if best < 0 {
			continue
		}
		// Lowercasing can in principle change a string's length, so the span
		// is clamped rather than trusted.
		end := min(best+length, len(line))
		return line, line[best:end]
	}
	return e.Excerpt(0), ""
}

func isHeading(line string) bool {
	_, ok := entry.SectionAt(line)
	return ok
}

// indexWord finds a needle that starts a word, so "go" matches "going" but
// not "algorithm". Both arguments must already be lowercase.
func indexWord(haystack, needle string) int {
	if needle == "" {
		return -1
	}
	for off := 0; off < len(haystack); {
		i := strings.Index(haystack[off:], needle)
		if i < 0 {
			return -1
		}
		i += off
		if i == 0 || !endsWord(haystack[:i]) {
			return i
		}
		off = i + 1
	}
	return -1
}

// endsWord reports whether the text ends in the middle of a word, which is
// where a match should not count.
func endsWord(s string) bool {
	r, size := utf8.DecodeLastRuneInString(s)
	if size == 0 {
		return false
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// Run streams the days that answer a query, newest first. Returning ErrStop
// from fn ends the search quietly.
func Run(j store.Journal, q Query, fn func(Match) error) error {
	return j.Walk(q.Since, q.Until, func(e entry.Entry) error {
		if e.IsEmpty() {
			return nil
		}
		m, ok := q.Match(e)
		if !ok {
			return nil
		}
		return fn(m)
	})
}

// All collects the matches, up to a limit. A limit of zero means all of them.
func All(j store.Journal, q Query, limit int) ([]Match, error) {
	var out []Match
	err := Run(j, q, func(m Match) error {
		out = append(out, m)
		if limit > 0 && len(out) >= limit {
			return ErrStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrStop) {
		return nil, err
	}
	return out, nil
}

// TagCount is a tag and how often it has been used.
type TagCount struct {
	Tag  string `json:"tag"`
	Days int    `json:"days"`
}

// Tags counts the tags across a range of days, most used first. A tag used
// twice on one day counts once: this is a count of days, not of mentions.
func Tags(j store.Journal, from, to entry.Date) ([]TagCount, error) {
	counts := map[string]int{}
	err := j.Walk(from, to, func(e entry.Entry) error {
		for _, t := range e.Tags() {
			counts[t]++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]TagCount, 0, len(counts))
	for tag, n := range counts {
		out = append(out, TagCount{Tag: tag, Days: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Days != out[j].Days {
			return out[i].Days > out[j].Days
		}
		return out[i].Tag < out[j].Tag
	})
	return out, nil
}

// token is one piece of a typed query, and whether it arrived in quotes.
type token struct {
	text   string
	quoted bool
}

// tokenize splits on whitespace, keeping quoted runs together. An unclosed
// quote runs to the end of the line rather than being an error — you are
// half-way through typing, not making a mistake.
func tokenize(s string) []token {
	var (
		out   []token
		cur   strings.Builder
		quote rune
		open  bool
	)
	flush := func() {
		if cur.Len() > 0 || open {
			out = append(out, token{text: cur.String(), quoted: open})
			cur.Reset()
		}
		open = false
	}

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				flush()
				continue
			}
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			flush()
			quote = r
			open = true
		case unicode.IsSpace(r):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}
