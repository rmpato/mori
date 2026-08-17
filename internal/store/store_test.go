package store

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/rmpato/mori/internal/entry"
)

func date(y int, m time.Month, d int) entry.Date {
	return entry.Date{Year: y, Month: m, Day: d}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	return New(filepath.Join(t.TempDir(), "journal"))
}

// write is the shorthand the range tests use to fill a journal.
func write(t *testing.T, s *Store, d entry.Date, body string) {
	t.Helper()
	if err := s.Put(entry.Entry{Date: d, Body: body}); err != nil {
		t.Fatalf("Put(%v): %v", d, err)
	}
}

func TestPathLayout(t *testing.T) {
	s := New("/journal")
	want := filepath.Join("/journal", "2026", "08", "2026-08-17.md")
	if got := s.Path(date(2026, time.August, 17)); got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

// A day you never wrote is a blank page, not an error. That is the honest
// answer to "what did I write on the third of May".
func TestGetMissingDayIsBlank(t *testing.T) {
	s := testStore(t)
	d := date(2026, time.May, 3)

	e, err := s.Get(d)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if e.Date != d {
		t.Errorf("Date = %v, want %v", e.Date, d)
	}
	if !e.IsEmpty() {
		t.Errorf("Body = %q, want empty", e.Body)
	}

	has, err := s.Has(d)
	if err != nil {
		t.Fatalf("Has: %v", err)
	}
	if has {
		t.Error("Has = true for a day that was never written")
	}
}

func TestPutGetRoundTrip(t *testing.T) {
	s := testStore(t)
	d := date(2026, time.August, 17)
	want := entry.Entry{
		Date: d,
		Mood: "calm",
		Body: "Slow morning. #go\n\n## 23:04\n\nWent back to it after dinner.",
	}

	if err := s.Put(want); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(d)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Mood != want.Mood || got.Body != want.Body {
		t.Errorf("round trip changed the page:\n got %#v\nwant %#v", got, want)
	}
	if tags := got.Tags(); !reflect.DeepEqual(tags, []string{"go"}) {
		t.Errorf("Tags = %v", tags)
	}
	if n := len(got.Sections()); n != 2 {
		t.Errorf("Sections = %d, want 2", n)
	}
}

// The file on disk has to stay something you'd be happy to open in any
// editor: no metadata you didn't ask for, and no date to disagree with.
func TestFileOnDiskIsPlainMarkdown(t *testing.T) {
	s := testStore(t)
	d := date(2026, time.August, 17)
	write(t, s, d, "a quiet day")

	raw, err := os.ReadFile(s.Path(d))
	if err != nil {
		t.Fatalf("reading the file back: %v", err)
	}
	if string(raw) != "a quiet day\n" {
		t.Errorf("file = %q, want just the writing", raw)
	}
}

// Emptying a page removes it, so the calendar never fills up with days you
// opened and didn't write in.
func TestPutEmptyRemovesTheFile(t *testing.T) {
	s := testStore(t)
	d := date(2026, time.August, 17)
	write(t, s, d, "something")

	if err := s.Put(entry.Entry{Date: d, Body: "   \n"}); err != nil {
		t.Fatalf("Put empty: %v", err)
	}
	if has, _ := s.Has(d); has {
		t.Error("the file survived being emptied")
	}
	// And the directories it lived in go with it.
	if _, err := os.Stat(filepath.Join(s.Root(), "2026")); !errors.Is(err, os.ErrNotExist) {
		t.Error("empty year directory was left behind")
	}
	// Emptying a day that was already empty is fine.
	if err := s.Put(entry.Entry{Date: d}); err != nil {
		t.Errorf("Put on an already-missing day: %v", err)
	}
}

// A mood with no writing is still something you said about the day.
func TestPutKeepsAMoodWithoutABody(t *testing.T) {
	s := testStore(t)
	d := date(2026, time.August, 17)

	if err := s.Put(entry.Entry{Date: d, Mood: "tired"}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(d)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Mood != "tired" {
		t.Errorf("Mood = %q, want %q", got.Mood, "tired")
	}
}

func TestPutRejectsADatelessPage(t *testing.T) {
	if err := testStore(t).Put(entry.Entry{Body: "x"}); err == nil {
		t.Error("Put accepted a page with no date")
	}
}

// A journal is nobody else's business.
func TestPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions")
	}
	s := testStore(t)
	d := date(2026, time.August, 17)
	write(t, s, d, "private")

	fi, err := os.Stat(s.Path(d))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := fi.Mode().Perm(); got != filePerm {
		t.Errorf("file mode = %o, want %o", got, filePerm)
	}

	di, err := os.Stat(filepath.Dir(s.Path(d)))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if got := di.Mode().Perm(); got != dirPerm {
		t.Errorf("directory mode = %o, want %o", got, dirPerm)
	}
}

func TestDates(t *testing.T) {
	s := testStore(t)
	all := []entry.Date{
		date(2025, time.December, 31),
		date(2026, time.January, 1),
		date(2026, time.August, 3),
		date(2026, time.August, 17),
		date(2026, time.September, 1),
	}
	for _, d := range all {
		write(t, s, d, "x")
	}

	tests := []struct {
		name     string
		from, to entry.Date
		want     []entry.Date
	}{
		{"everything", entry.Date{}, entry.Date{}, all},
		{"open start", entry.Date{}, date(2026, time.January, 1), all[:2]},
		{"open end", date(2026, time.August, 3), entry.Date{}, all[2:]},
		{"one month", date(2026, time.August, 1), date(2026, time.August, 31), all[2:4]},
		{"bounds are inclusive", date(2026, time.August, 17), date(2026, time.August, 17), all[3:4]},
		{"a gap with nothing in it", date(2026, time.March, 1), date(2026, time.April, 1), nil},
		{"backwards bounds are read charitably", date(2026, time.August, 31), date(2026, time.August, 1), all[2:4]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Dates(tt.from, tt.to)
			if err != nil {
				t.Fatalf("Dates: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Dates = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatesOnAJournalThatDoesNotExistYet(t *testing.T) {
	got, err := testStore(t).Dates(entry.Date{}, entry.Date{})
	if err != nil {
		t.Fatalf("Dates: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Dates = %v, want nothing", got)
	}
}

// The tree belongs to the user, who may keep other things in it.
func TestDatesIgnoresStrangers(t *testing.T) {
	s := testStore(t)
	write(t, s, date(2026, time.August, 17), "x")

	month := filepath.Join(s.Root(), "2026", "08")
	for _, name := range []string{"notes.txt", "README.md", "2026-8-1.md", "scratch.md"} {
		if err := os.WriteFile(filepath.Join(month, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(s.Root(), "attachments"), 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := s.Dates(entry.Date{}, entry.Date{})
	if err != nil {
		t.Fatalf("Dates: %v", err)
	}
	if want := []entry.Date{date(2026, time.August, 17)}; !reflect.DeepEqual(got, want) {
		t.Errorf("Dates = %v, want %v", got, want)
	}
}

// A file filed under the wrong month is not to be trusted about its own date.
func TestDatesIgnoresMisfiledPages(t *testing.T) {
	s := testStore(t)
	write(t, s, date(2026, time.August, 17), "x")

	misfiled := filepath.Join(s.Root(), "2026", "08", "2026-09-02.md")
	if err := os.WriteFile(misfiled, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := s.Dates(entry.Date{}, entry.Date{})
	if err != nil {
		t.Fatalf("Dates: %v", err)
	}
	if want := []entry.Date{date(2026, time.August, 17)}; !reflect.DeepEqual(got, want) {
		t.Errorf("Dates = %v, want %v", got, want)
	}
}

// Looking back means newest first.
func TestWalkGoesBackwards(t *testing.T) {
	s := testStore(t)
	for _, d := range []entry.Date{
		date(2026, time.August, 1),
		date(2026, time.August, 10),
		date(2026, time.August, 17),
	} {
		write(t, s, d, d.String())
	}

	var seen []string
	if err := s.Walk(entry.Date{}, entry.Date{}, func(e entry.Entry) error {
		seen = append(seen, e.Body)
		return nil
	}); err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := []string{"2026-08-17", "2026-08-10", "2026-08-01"}
	if !reflect.DeepEqual(seen, want) {
		t.Errorf("Walk visited %v, want %v", seen, want)
	}
}

func TestWalkStopsQuietly(t *testing.T) {
	s := testStore(t)
	for day := 1; day <= 5; day++ {
		write(t, s, date(2026, time.August, day), "x")
	}

	var n int
	err := s.Walk(entry.Date{}, entry.Date{}, func(entry.Entry) error {
		n++
		if n == 2 {
			return ErrStop
		}
		return nil
	})
	if err != nil {
		t.Errorf("Walk returned %v, want nil for ErrStop", err)
	}
	if n != 2 {
		t.Errorf("Walk visited %d pages, want 2", n)
	}
}

func TestWalkPassesRealErrorsBack(t *testing.T) {
	s := testStore(t)
	write(t, s, date(2026, time.August, 17), "x")

	boom := errors.New("boom")
	if err := s.Walk(entry.Date{}, entry.Date{}, func(entry.Entry) error {
		return boom
	}); !errors.Is(err, boom) {
		t.Errorf("Walk returned %v, want %v", err, boom)
	}
}

func TestDefaultRoot(t *testing.T) {
	t.Setenv("MORI_DIR", "")
	t.Setenv("MORI_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")

	t.Run("MORI_DIR wins", func(t *testing.T) {
		t.Setenv("MORI_DIR", "/somewhere/pages")
		if got, _ := DefaultRoot(); got != "/somewhere/pages" {
			t.Errorf("DefaultRoot = %q", got)
		}
	})

	t.Run("then MORI_HOME", func(t *testing.T) {
		t.Setenv("MORI_HOME", "/somewhere/mori")
		want := filepath.Join("/somewhere/mori", "journal")
		if got, _ := DefaultRoot(); got != want {
			t.Errorf("DefaultRoot = %q, want %q", got, want)
		}
	})

	t.Run("then XDG_DATA_HOME", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/xdg")
		want := filepath.Join("/xdg", "mori", "journal")
		if got, _ := DefaultRoot(); got != want {
			t.Errorf("DefaultRoot = %q, want %q", got, want)
		}
	})

	t.Run("otherwise the usual home", func(t *testing.T) {
		got, err := DefaultRoot()
		if err != nil {
			t.Fatalf("DefaultRoot: %v", err)
		}
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".local", "share", "mori", "journal")
		if got != want {
			t.Errorf("DefaultRoot = %q, want %q", got, want)
		}
	})
}

// An interrupted save must never be able to leave half a page behind. The
// rename is what guarantees it; this checks nothing is left lying around.
func TestWritesLeaveNoTemporaryFiles(t *testing.T) {
	s := testStore(t)
	d := date(2026, time.August, 17)
	write(t, s, d, "first")
	write(t, s, d, "second")

	ents, err := os.ReadDir(filepath.Dir(s.Path(d)))
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != "2026-08-17.md" {
		var names []string
		for _, e := range ents {
			names = append(names, e.Name())
		}
		t.Errorf("directory holds %v, want just the page", names)
	}
}
