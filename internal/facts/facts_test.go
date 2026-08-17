package facts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rmpato/mori/internal/entry"
)

var aug17 = entry.Date{Year: 2026, Month: time.August, Day: 17}

// tukiFileWith writes a tasks.json the way tuki would, and returns its path.
func tukiFileWith(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A real tuki file, as tuki writes it.
const sample = `{
  "version": 1,
  "next_id": 6,
  "settings": {"celebrate": true, "judge": true},
  "tasks": [
    {"id": 1, "text": "Finish Go project", "tag": "work", "done": true,
     "created_at": "2026-08-15T09:00:00Z", "done_at": "2026-08-17T18:20:00Z"},
    {"id": 2, "text": "Go to gym", "tag": "home", "done": true,
     "created_at": "2026-08-17T08:00:00Z", "done_at": "2026-08-17T19:00:00Z"},
    {"id": 3, "text": "Edit photos", "tag": "photography", "done": false,
     "created_at": "2026-08-17T08:05:00Z"},
    {"id": 4, "text": "Fix API tests", "tag": "work", "done": true,
     "created_at": "2026-08-10T10:00:00Z", "done_at": "2026-08-12T11:00:00Z"},
    {"id": 5, "text": "Renew passport", "tag": "misc", "done": false,
     "created_at": "2026-01-02T10:00:00Z", "due": "2026-08-14T00:00:00Z"}
  ]
}`

func TestDay(t *testing.T) {
	src := Tuki(tukiFileWith(t, sample))

	snap, err := src.Day(aug17)
	if err != nil {
		t.Fatalf("Day: %v", err)
	}
	if snap.Source != "tuki" {
		t.Errorf("Source = %q", snap.Source)
	}

	// Finished on the day itself — not before it, not after it.
	if got := texts(snap.Done); !equal(got, []string{"Finish Go project", "Go to gym"}) {
		t.Errorf("Done = %v", got)
	}
	// Still open, and on your plate that day: made that day, or overdue by it.
	if got := texts(snap.Todo); !equal(got, []string{"Edit photos", "Renew passport"}) {
		t.Errorf("Todo = %v", got)
	}
	// Tags come across, normalised the way mori spells them.
	if snap.Done[0].Tag != "work" || snap.Todo[0].Tag != "photography" {
		t.Errorf("tags = %q, %q", snap.Done[0].Tag, snap.Todo[0].Tag)
	}
}

// A task finished on another day belongs to that day, not this one.
func TestDayIsAboutOneDay(t *testing.T) {
	src := Tuki(tukiFileWith(t, sample))

	snap, err := src.Day(entry.Date{Year: 2026, Month: time.August, Day: 12})
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(snap.Done); !equal(got, []string{"Fix API tests"}) {
		t.Errorf("Done on the 12th = %v", got)
	}
}

// Without this, every task you have ever left open would follow you backwards
// through the whole journal.
func TestOpenTasksDontFollowYouBackwards(t *testing.T) {
	src := Tuki(tukiFileWith(t, sample))

	// A day before the open tasks were made, and before the overdue one was due.
	snap, err := src.Day(entry.Date{Year: 2026, Month: time.August, Day: 13})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Todo) != 0 {
		t.Errorf("Todo on the 13th = %v, want nothing", texts(snap.Todo))
	}
}

// No tuki, no context, no complaint.
func TestAMissingTukiIsNotAProblem(t *testing.T) {
	src := Tuki(filepath.Join(t.TempDir(), "nothing here", "tasks.json"))

	snap, err := src.Day(aug17)
	if err != nil {
		t.Fatalf("Day: %v", err)
	}
	if !snap.IsEmpty() {
		t.Errorf("Snapshot = %+v, want nothing", snap)
	}
}

func TestAvailable(t *testing.T) {
	path := tukiFileWith(t, sample)
	if !Available(path) {
		t.Error("Available = false for a file that's there")
	}
	if Available(filepath.Join(t.TempDir(), "no.json")) {
		t.Error("Available = true for a file that isn't")
	}
}

// A tuki that has moved on is something mori says it doesn't understand,
// rather than something it guesses at.
func TestAFutureTukiIsRefusedClearly(t *testing.T) {
	src := Tuki(tukiFileWith(t, `{"version": 99, "tasks": []}`))

	if _, err := src.Day(aug17); err == nil {
		t.Fatal("Day read a file from the future")
	} else if !strings.Contains(err.Error(), "newer") {
		t.Errorf("error = %v, want it to say why", err)
	}
}

func TestNonsenseIsAnError(t *testing.T) {
	src := Tuki(tukiFileWith(t, "this is not JSON"))
	if _, err := src.Day(aug17); err == nil {
		t.Error("Day accepted a file that isn't a tuki file")
	}
}

// mori has to keep working when tuki's file grows fields mori has never
// heard of.
func TestUnknownFieldsAreIgnored(t *testing.T) {
	src := Tuki(tukiFileWith(t, `{
		"version": 1,
		"projects": [{"name": "zine"}],
		"tasks": [{"text": "Go to gym", "done": true, "done_at": "2026-08-17T19:00:00Z",
		           "priority": "high", "subtasks": []}]
	}`))

	snap, err := src.Day(aug17)
	if err != nil {
		t.Fatalf("Day: %v", err)
	}
	if got := texts(snap.Done); !equal(got, []string{"Go to gym"}) {
		t.Errorf("Done = %v", got)
	}
}

func TestTukiPath(t *testing.T) {
	t.Setenv("TUKI_FILE", "")
	t.Setenv("TUKI_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")

	t.Run("TUKI_FILE wins", func(t *testing.T) {
		t.Setenv("TUKI_FILE", "/somewhere/tasks.json")
		if got, _ := TukiPath(); got != "/somewhere/tasks.json" {
			t.Errorf("TukiPath = %q", got)
		}
	})
	t.Run("then TUKI_HOME", func(t *testing.T) {
		t.Setenv("TUKI_HOME", "/somewhere/tuki")
		want := filepath.Join("/somewhere/tuki", "tasks.json")
		if got, _ := TukiPath(); got != want {
			t.Errorf("TukiPath = %q, want %q", got, want)
		}
	})
	t.Run("otherwise the usual home", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".local", "share", "tuki", "tasks.json")
		if got, _ := TukiPath(); got != want {
			t.Errorf("TukiPath = %q, want %q", got, want)
		}
	})
}

// ------------------------------------------------------------- template ---

func TestTemplate(t *testing.T) {
	snap := Snapshot{
		Source: "tuki",
		Done:   []Item{{Text: "Finish Go project"}, {Text: "Read GitOps chapter"}},
		Todo:   []Item{{Text: "Edit photography photos"}},
	}
	got := Template(aug17, snap)

	for _, want := range []string{
		"# August 17, 2026",
		"## Today",
		"## Things I did",
		"- Finish Go project",
		"- Read GitOps chapter",
		"## Things I didn't get to",
		"- Edit photography photos",
		"## Notes",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the template is missing %q:\n%s", want, got)
		}
	}
}

// The whole point of the two tools is that one holds what you did and the
// other holds what it was like. mori supplies headings and your own task
// text — never a sentence about how the day went.
func TestTemplateWritesNoProse(t *testing.T) {
	snap := Snapshot{Done: []Item{{Text: "Go to gym"}}, Todo: []Item{{Text: "Edit photos"}}}
	got := Template(aug17, snap)

	for _, line := range strings.Split(got, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "", strings.HasPrefix(line, "#"):
			// A heading, which is scaffolding.
		case strings.HasPrefix(line, "- "):
			// A task, in your own words, copied exactly.
			text := strings.TrimPrefix(line, "- ")
			if text != "Go to gym" && text != "Edit photos" {
				t.Errorf("the template invented a bullet: %q", text)
			}
		default:
			t.Errorf("the template wrote prose: %q", line)
		}
	}
}

// A day with nothing in it still gives you somewhere to write.
func TestTemplateForAnEmptyDay(t *testing.T) {
	got := Template(aug17, Snapshot{Source: "tuki"})

	for _, want := range []string{"# August 17, 2026", "## Today", "## Notes"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Things I did", "didn't get to"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("an empty day got a %q heading:\n%s", unwanted, got)
		}
	}
}

func texts(items []Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Text)
	}
	return out
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
