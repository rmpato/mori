package entry

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Entry is one day's page.
//
// There is no ID, no title, and no created-at: the date is the identity and
// the file has the timestamps. Everything else about an entry — its tags, the
// sittings it was written in — is read out of the prose rather than stored
// beside it, so there is only ever one thing to keep true.
type Entry struct {
	Date Date
	Mood string // optional, one short word; empty most of the time
	Body string // Markdown, as it was typed

	// extra holds frontmatter keys mori doesn't know about, in file order, so
	// a future mori — or your own editing — never loses anything.
	extra []field
}

type field struct{ Key, Value string }

// New is an empty page for a day.
func New(d Date) Entry { return Entry{Date: d} }

// IsEmpty reports whether there is nothing on the page. Section headings on
// their own don't count as writing: a day you opened, stamped, and left is
// still a day you didn't write.
func (e Entry) IsEmpty() bool {
	for line := range strings.Lines(e.Body) {
		if strings.TrimSpace(line) == "" || isSectionHeading(line) {
			continue
		}
		return false
	}
	return true
}

// Words is roughly how much you wrote, ignoring section headings.
func (e Entry) Words() int {
	var n int
	for line := range strings.Lines(e.Body) {
		if isSectionHeading(line) {
			continue
		}
		n += len(strings.Fields(line))
	}
	return n
}

// Excerpt is the first real line of the page, cut to width, for lists and
// search results.
func (e Entry) Excerpt(width int) string {
	for line := range strings.Lines(e.Body) {
		line = strings.TrimSpace(line)
		if line == "" || isSectionHeading(line) {
			continue
		}
		// Markdown headings read better in a list without their hashes.
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(strings.TrimLeft(line, ">-*+ "))
		if line == "" {
			continue
		}
		return truncate(line, width)
	}
	return ""
}

func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return strings.TrimRight(string(r[:width-1]), " ") + "…"
}

// ---------------------------------------------------------------- tags -----

// tagRe finds a hashtag in prose. The tag has to start with a letter, digit,
// or underscore, which is what keeps a "## 23:04" section heading and a
// "# Heading" from being read as tags.
var tagRe = regexp.MustCompile(`(^|\s)#(\w[\w-]*)`)

// NormalizeTag lowercases a tag, drops a leading '#', and strips anything
// that isn't a letter, digit, dash, or underscore — the same rules tuki uses,
// so the two tools agree on what "#photography" means.
//
// Unlike tuki, an empty result stays empty: a day with no tags has no tags,
// and there is no catch-all bucket to fall into.
func NormalizeTag(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "#")
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return -1
	}, s)
}

// Tags are the hashtags in the page, deduplicated, in the order you wrote
// them. They live in the prose and are parsed on the way out, so tagging a
// day is just writing the way you already write.
func (e Entry) Tags() []string {
	matches := tagRe.FindAllStringSubmatch(e.Body, -1)
	if matches == nil {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := NormalizeTag(m[2])
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	return tags
}

// HasTag reports whether the page carries a tag.
func (e Entry) HasTag(tag string) bool {
	tag = NormalizeTag(tag)
	if tag == "" {
		return false
	}
	for _, t := range e.Tags() {
		if t == tag {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------ sections -----

// Section is one sitting: a stretch of the day's page written at one time.
type Section struct {
	At   string `json:"at,omitempty"` // "23:04", empty for the block the day opens with
	Body string `json:"body"`
}

// sectionRe matches a section heading and nothing else. The format is
// deliberately strict so that "## Things I did" — and any other Markdown you
// write — stays prose.
var sectionRe = regexp.MustCompile(`^##[ \t]+(\d{1,2}):(\d{2})[ \t]*$`)

func isSectionHeading(line string) bool { _, ok := headingTime(line); return ok }

// SectionAt reports the time a line stamps, if it is a section heading. It is
// here so that anything rendering a page can recognise the heading without
// re-deriving the format.
func SectionAt(line string) (string, bool) { return headingTime(line) }

// headingTime returns the normalised "HH:MM" a heading line carries.
func headingTime(line string) (string, bool) {
	m := sectionRe.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
	if m == nil {
		return "", false
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	if h > 23 || min > 59 {
		return "", false
	}
	return fmt.Sprintf("%02d:%02d", h, min), true
}

// SectionHeading is the line mori writes when you come back to a day later.
func SectionHeading(t time.Time) string { return "## " + t.Format("15:04") }

// TrimTrailingEmptySection drops a section heading with nothing underneath
// it. mori stamps the page when you sit back down, and if you then don't
// write anything, the sitting didn't happen and shouldn't leave a mark.
func TrimTrailingEmptySection(body string) string {
	trimmed := strings.TrimRight(body, " \t\n")
	i := strings.LastIndexByte(trimmed, '\n')
	if !isSectionHeading(trimmed[i+1:]) {
		return body
	}
	return strings.TrimRight(trimmed[:i+1], " \t\n")
}

// Sections splits the page into the sittings it was written in.
//
// This is a reading of the body, never a second kind of object: Body stays
// the one true thing, and a day written in one go comes back as a single
// section with no timestamp on it.
func (e Entry) Sections() []Section {
	var (
		out  []Section
		cur  = Section{}
		body strings.Builder
	)
	flush := func() {
		cur.Body = strings.TrimSpace(body.String())
		// Drop the opening block when the page starts with a heading, but keep
		// a stamped section that happens to be empty — you meant to make it.
		if cur.At != "" || cur.Body != "" {
			out = append(out, cur)
		}
		body.Reset()
	}

	for line := range strings.Lines(e.Body) {
		if at, ok := headingTime(line); ok {
			flush()
			cur = Section{At: at}
			continue
		}
		body.WriteString(line)
	}
	flush()
	return out
}

// ---------------------------------------------------------------- mood -----

// NormalizeMood keeps a mood to one short lowercase word, because that is the
// entire feature. There is no scale, no score, and nothing that could be
// averaged.
func NormalizeMood(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
		s = s[:i]
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || r == '-' {
			return r
		}
		return -1
	}, s)
}
