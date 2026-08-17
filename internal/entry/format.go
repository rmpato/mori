package entry

import (
	"strings"
)

// A mori entry on disk is a Markdown file, and in the common case that is all
// it is: no header, no metadata, nothing between you and the first sentence.
//
//	Today was actually pretty good.
//
//	I finally started that little Go project. #go
//
// Frontmatter appears only when there is something to put in it:
//
//	---
//	mood: calm
//	---
//
//	Today was actually pretty good.
//
// The date is deliberately absent — it is the filename, and duplicating it
// would only create something to disagree with.
const fence = "---"

// Parse reads a day's file. It is deliberately forgiving: anything it can't
// make sense of is treated as prose rather than reported as an error, because
// the worst outcome for a journal is refusing to show you what you wrote.
func Parse(d Date, raw []byte) Entry {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	e := Entry{Date: d}

	rest, fields, ok := splitFrontmatter(text)
	if ok {
		for _, f := range fields {
			switch f.Key {
			case "mood":
				e.Mood = NormalizeMood(f.Value)
			default:
				e.extra = append(e.extra, f)
			}
		}
		text = rest
	}

	e.Body = strings.TrimSpace(text)
	return e
}

// splitFrontmatter peels a leading --- block off the text. It reports false
// when there isn't one, or when the block is never closed — in which case the
// dashes were part of what you wrote.
func splitFrontmatter(text string) (rest string, fields []field, ok bool) {
	if !strings.HasPrefix(text, fence+"\n") {
		return text, nil, false
	}
	body := text[len(fence)+1:]
	end := strings.Index(body, "\n"+fence)
	if end < 0 {
		return text, nil, false
	}
	head := body[:end]
	rest = body[end+len(fence)+1:]
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[i+1:] // drop whatever trails the closing fence
	} else {
		rest = ""
	}

	for line := range strings.Lines(head) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		fields = append(fields, field{
			Key:   strings.ToLower(strings.TrimSpace(key)),
			Value: strings.TrimSpace(value),
		})
	}
	return rest, fields, true
}

// Format renders the entry back to the bytes that go on disk. Parse and
// Format round-trip: anything mori didn't understand comes back out in the
// order it went in.
func (e Entry) Format() []byte {
	var b strings.Builder

	fields := make([]field, 0, len(e.extra)+1)
	if e.Mood != "" {
		fields = append(fields, field{Key: "mood", Value: e.Mood})
	}
	fields = append(fields, e.extra...)

	if len(fields) > 0 {
		b.WriteString(fence + "\n")
		for _, f := range fields {
			b.WriteString(f.Key + ": " + f.Value + "\n")
		}
		b.WriteString(fence + "\n\n")
	}

	body := strings.TrimSpace(e.Body)
	if body != "" {
		b.WriteString(body)
		b.WriteString("\n")
	}
	return []byte(b.String())
}
