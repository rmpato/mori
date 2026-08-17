package entry

import (
	"strings"
	"testing"
	"time"
)

var aug17 = Date{Year: 2026, Month: time.August, Day: 17}

func TestParsePlainPage(t *testing.T) {
	raw := "Today was actually pretty good.\n\nI started that little Go project. #go\n"
	e := Parse(aug17, []byte(raw))

	if e.Date != aug17 {
		t.Errorf("Date = %v", e.Date)
	}
	if e.Mood != "" {
		t.Errorf("Mood = %q, want empty", e.Mood)
	}
	if want := strings.TrimSpace(raw); e.Body != want {
		t.Errorf("Body = %q, want %q", e.Body, want)
	}
}

func TestParseFrontmatter(t *testing.T) {
	raw := "---\nmood: Calm\n---\n\nToday was pretty good.\n"
	e := Parse(aug17, []byte(raw))

	if e.Mood != "calm" {
		t.Errorf("Mood = %q, want %q", e.Mood, "calm")
	}
	if e.Body != "Today was pretty good." {
		t.Errorf("Body = %q", e.Body)
	}
}

func TestParseIsForgiving(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		body string
	}{
		{
			name: "an unclosed fence is prose, not a broken header",
			raw:  "---\nmood: calm\n\nsome writing\n",
			body: "---\nmood: calm\n\nsome writing",
		},
		{
			name: "a horizontal rule mid-page is left alone",
			raw:  "morning\n\n---\n\nevening\n",
			body: "morning\n\n---\n\nevening",
		},
		{
			name: "windows line endings",
			raw:  "---\r\nmood: calm\r\n---\r\n\r\nhello\r\n",
			body: "hello",
		},
		{
			name: "an empty file",
			raw:  "",
			body: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Parse(aug17, []byte(tt.raw)).Body; got != tt.body {
				t.Errorf("Body = %q, want %q", got, tt.body)
			}
		})
	}
}

func TestFormatOmitsEmptyFrontmatter(t *testing.T) {
	e := Entry{Date: aug17, Body: "a quiet day"}
	if got := string(e.Format()); got != "a quiet day\n" {
		t.Errorf("Format = %q, want no frontmatter at all", got)
	}
}

func TestFormatWritesMood(t *testing.T) {
	e := Entry{Date: aug17, Mood: "calm", Body: "a quiet day"}
	want := "---\nmood: calm\n---\n\na quiet day\n"
	if got := string(e.Format()); got != want {
		t.Errorf("Format = %q, want %q", got, want)
	}
}

// The date is the filename. Writing it into the file too would only create
// something to disagree with.
func TestFormatNeverWritesTheDate(t *testing.T) {
	e := Entry{Date: aug17, Mood: "calm", Body: "x"}
	if strings.Contains(string(e.Format()), "2026-08-17") {
		t.Error("Format wrote the date into the file")
	}
}

func TestRoundTrip(t *testing.T) {
	for _, e := range []Entry{
		{Date: aug17, Body: "a quiet day"},
		{Date: aug17, Mood: "calm", Body: "a quiet day"},
		{Date: aug17, Body: "morning\n\n## 23:04\n\nevening"},
		{Date: aug17, Body: "trailing whitespace   "},
		{Date: aug17},
	} {
		got := Parse(e.Date, e.Format())
		if got.Mood != e.Mood {
			t.Errorf("round trip lost the mood: %q -> %q", e.Mood, got.Mood)
		}
		if want := strings.TrimSpace(e.Body); got.Body != want {
			t.Errorf("round trip changed the body: %q -> %q", want, got.Body)
		}
	}
}

// A future mori, or your own editing, may put keys in the frontmatter this
// one has never heard of. Losing them would be losing your writing.
func TestUnknownFrontmatterSurvives(t *testing.T) {
	raw := "---\nmood: calm\nweather: rain\nlocation: bariloche\n---\n\nhello\n"
	e := Parse(aug17, []byte(raw))

	out := string(e.Format())
	for _, want := range []string{"weather: rain", "location: bariloche", "mood: calm"} {
		if !strings.Contains(out, want) {
			t.Errorf("Format lost %q:\n%s", want, out)
		}
	}
	if got := Parse(aug17, []byte(out)); got.Body != "hello" {
		t.Errorf("Body = %q after a second trip", got.Body)
	}
}
