package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/store"
	"github.com/rmpato/mori/internal/ui"
)

// journal is a store in a temp directory, plus the runner that points mori at
// it. Output goes to a buffer rather than a terminal, so every command under
// test takes its own no-tty path — which is the one scripts get.
func journal(t *testing.T) (*store.Store, func(args ...string) (string, error)) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "journal")
	s := store.New(dir)

	// Point everything mori reads at somewhere that doesn't exist yet, so a
	// test can never pick up the config or the task list of whoever is
	// running it.
	t.Setenv("MORI_CONFIG", filepath.Join(root, "config.json"))
	t.Setenv("TUKI_FILE", filepath.Join(root, "tasks.json"))

	run := func(args ...string) (string, error) {
		var buf bytes.Buffer
		root := NewRoot()
		root.SetOut(&buf)
		root.SetErr(&buf)
		// The same normalisation Execute does, so the tests exercise the
		// arguments a person actually types.
		root.SetArgs(normalizeArgs(append([]string{"--dir", dir}, args...)))
		err := root.Execute()
		return buf.String(), err
	}
	return s, run
}

func write(t *testing.T, s *store.Store, d entry.Date, e entry.Entry) {
	t.Helper()
	e.Date = d
	if err := s.Put(e); err != nil {
		t.Fatalf("Put: %v", err)
	}
}

func TestPath(t *testing.T) {
	s, run := journal(t)

	out, err := run("path")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if got := strings.TrimSpace(out); got != s.Root() {
		t.Errorf("path = %q, want %q", got, s.Root())
	}

	// A date gives you the file, whether or not it exists yet — which is what
	// makes `$EDITOR "$(mori path yesterday)"` work.
	out, err = run("path", "2026-08-17")
	if err != nil {
		t.Fatalf("path <date>: %v", err)
	}
	want := filepath.Join(s.Root(), "2026", "08", "2026-08-17.md")
	if got := strings.TrimSpace(out); got != want {
		t.Errorf("path <date> = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err == nil {
		t.Error("asking for a path created the file")
	}
}

// Piping a page somewhere should give you the page, not a drawing of it.
func TestShowPipesRawMarkdown(t *testing.T) {
	s, run := journal(t)
	d := entry.Date{Year: 2026, Month: time.August, Day: 17}
	write(t, s, d, entry.Entry{Body: "Slow morning. #go"})

	out, err := run("show", "2026-08-17")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if out != "Slow morning. #go\n" {
		t.Errorf("show = %q, want the file's own contents", out)
	}
}

func TestShowSpeaksFriendlyDates(t *testing.T) {
	s, run := journal(t)
	yesterday := entry.Today().Add(-1)
	write(t, s, yesterday, entry.Entry{Body: "yesterday's page"})

	for _, arg := range []string{"yesterday", "-1d", yesterday.String()} {
		out, err := run("show", arg)
		if err != nil {
			t.Fatalf("show %s: %v", arg, err)
		}
		if !strings.Contains(out, "yesterday's page") {
			t.Errorf("show %s = %q", arg, out)
		}
	}
}

func TestShowRejectsADateItCannotRead(t *testing.T) {
	_, run := journal(t)
	if _, err := run("show", "someday"); err == nil {
		t.Error("show accepted a date it can't possibly resolve")
	}
}

// A day you never wrote prints nothing down a pipe, so `mori show | wc -c`
// says zero rather than describing the absence.
func TestShowMissingDayIsSilentDownAPipe(t *testing.T) {
	_, run := journal(t)
	out, err := run("show", "2020-01-01")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if out != "" {
		t.Errorf("show = %q, want nothing", out)
	}
}

func TestTodayAndBareMoriAgree(t *testing.T) {
	s, run := journal(t)
	write(t, s, entry.Today(), entry.Entry{Body: "today's page"})

	today, err := run("today")
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	bare, err := run()
	if err != nil {
		t.Fatalf("mori: %v", err)
	}
	if today != bare {
		t.Errorf("`mori` = %q but `mori today` = %q", bare, today)
	}
	if !strings.Contains(today, "today's page") {
		t.Errorf("today = %q", today)
	}
}

func TestJSON(t *testing.T) {
	s, run := journal(t)
	d := entry.Date{Year: 2026, Month: time.August, Day: 17}
	write(t, s, d, entry.Entry{
		Mood: "calm",
		Body: "Slow morning. #go\n\n## 23:04\n\nBack to it. #photography",
	})

	out, err := run("show", "2026-08-17", "--json")
	if err != nil {
		t.Fatalf("show --json: %v", err)
	}

	var got struct {
		Date     string   `json:"date"`
		Mood     string   `json:"mood"`
		Body     string   `json:"body"`
		Tags     []string `json:"tags"`
		Words    int      `json:"words"`
		Sections []struct {
			At   string `json:"at"`
			Body string `json:"body"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}

	if got.Date != "2026-08-17" {
		t.Errorf("date = %q", got.Date)
	}
	if got.Mood != "calm" {
		t.Errorf("mood = %q", got.Mood)
	}
	if want := []string{"go", "photography"}; !equal(got.Tags, want) {
		t.Errorf("tags = %v, want %v", got.Tags, want)
	}
	if len(got.Sections) != 2 || got.Sections[1].At != "23:04" {
		t.Errorf("sections = %+v", got.Sections)
	}
	if got.Words == 0 {
		t.Error("words = 0")
	}
}

// The styled path is the one a person sees, and a buffer is never a terminal,
// so it has to be exercised directly rather than through a command.
func TestStyledOutputCarriesThePage(t *testing.T) {
	var buf bytes.Buffer
	e := &env{
		out:   &buf,
		w:     &buf,
		theme: ui.New(true),
		tty:   true,
		now:   time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC),
	}
	page := entry.Entry{
		Date: entry.Date{Year: 2026, Month: time.August, Day: 17},
		Mood: "calm",
		Body: "Slow morning.\n\n## 23:04\n\nBack to it.",
	}
	e.writeEntry(page, false)

	out := buf.String()
	for _, want := range []string{"Monday", "August 17, 2026", "calm", "Slow morning.", "Back to it.", "23:04"} {
		if !strings.Contains(out, want) {
			t.Errorf("styled output is missing %q:\n%s", want, out)
		}
	}
	// The raw section heading is rendered as a time, not left as Markdown.
	if strings.Contains(out, "## 23:04") {
		t.Errorf("styled output shows the raw heading:\n%s", out)
	}
}

// An empty day says so, gently, rather than printing nothing at a person.
func TestStyledEmptyDaySaysSo(t *testing.T) {
	var buf bytes.Buffer
	e := &env{out: &buf, w: &buf, theme: ui.New(true), tty: true, now: time.Now()}
	e.writeEntry(entry.New(entry.Date{Year: 2026, Month: time.August, Day: 17}), false)

	if !strings.Contains(buf.String(), "nothing written here") {
		t.Errorf("an empty day printed %q", buf.String())
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
