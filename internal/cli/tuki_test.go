package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rmpato/mori/internal/entry"
)

// tukiJournal is a journal plus a tuki file with today's tasks in it, both in
// temp directories, with the environment pointed at them.
func tukiJournal(t *testing.T, tasks string) func(args ...string) (string, error) {
	t.Helper()

	_, run := journal(t)

	dir := t.TempDir()
	tukiFile := filepath.Join(dir, "tasks.json")
	if tasks != "" {
		if err := os.WriteFile(tukiFile, []byte(tasks), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Set after journal, which points these at nothing by default.
	t.Setenv("TUKI_FILE", tukiFile)
	return run
}

// tasksToday is a tuki file with one task finished today and one still open.
func tasksToday(t *testing.T) string {
	t.Helper()
	today := entry.Today().Time().Format("2006-01-02")
	return `{"version": 1, "tasks": [
		{"text": "Finish Go project", "tag": "work", "done": true,
		 "created_at": "` + today + `T09:00:00Z", "done_at": "` + today + `T18:00:00Z"},
		{"text": "Edit photos", "tag": "photography", "done": false,
		 "created_at": "` + today + `T09:00:00Z"}
	]}`
}

func TestFromTuki(t *testing.T) {
	run := tukiJournal(t, tasksToday(t))

	out, err := run("today", "--from-tuki")
	if err != nil {
		t.Fatalf("--from-tuki: %v", err)
	}
	for _, want := range []string{
		"## Today",
		"## Things I did",
		"- Finish Go project",
		"## Things I didn't get to",
		"- Edit photos",
		"## Notes",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// Printing a scaffold doesn't write one.
func TestFromTukiPrintsWithoutWriting(t *testing.T) {
	run := tukiJournal(t, tasksToday(t))

	if _, err := run("today", "--from-tuki"); err != nil {
		t.Fatal(err)
	}
	out, err := run("today")
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("today's page is %q, want it still blank", out)
	}
}

func TestFromTukiWrite(t *testing.T) {
	run := tukiJournal(t, tasksToday(t))

	if _, err := run("today", "--from-tuki", "--write"); err != nil {
		t.Fatalf("--write: %v", err)
	}
	out, err := run("today")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "## Things I did") {
		t.Errorf("today's page is %q", out)
	}
}

// A starting point is only a starting point when there is nothing there yet.
func TestFromTukiNeverOverwritesWriting(t *testing.T) {
	run := tukiJournal(t, tasksToday(t))

	if _, err := run("today", "--from-tuki", "--write"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("today", "--from-tuki", "--write"); err == nil {
		t.Error("--write went over the top of a page that already had writing")
	}
}

func TestFromTukiWithoutTuki(t *testing.T) {
	run := tukiJournal(t, "") // no tuki file at all

	if _, err := run("today", "--from-tuki"); err == nil {
		t.Error("--from-tuki worked without tuki installed")
	}
}

func TestConfig(t *testing.T) {
	run := tukiJournal(t, tasksToday(t))

	out, err := run("config", "--json")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	var got struct {
		Config  string `json:"config"`
		Journal string `json:"journal"`
		Tuki    struct {
			Enabled   bool   `json:"enabled"`
			Installed bool   `json:"installed"`
			File      string `json:"file"`
			Write     bool   `json:"write"`
		} `json:"tuki"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}

	if !got.Tuki.Installed || !got.Tuki.Enabled {
		t.Errorf("tuki = %+v, want it found and on", got.Tuki)
	}
	// Not a setting. Not now, not behind a flag.
	if got.Tuki.Write {
		t.Error("config reports that mori writes to tuki")
	}
}

func TestConfigWithoutTuki(t *testing.T) {
	run := tukiJournal(t, "")

	out, err := run("config", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Tuki struct {
			Enabled   bool `json:"enabled"`
			Installed bool `json:"installed"`
		} `json:"tuki"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Tuki.Installed || got.Tuki.Enabled {
		t.Errorf("tuki = %+v, want it absent and off", got.Tuki)
	}
}

// Turning the integration off means mori doesn't read tuki even when it's
// sitting right there.
func TestTukiCanBeTurnedOff(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfg, []byte(`{"tuki": {"enabled": false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tukiFile := filepath.Join(dir, "tasks.json")
	if err := os.WriteFile(tukiFile, []byte(tasksToday(t)), 0o600); err != nil {
		t.Fatal(err)
	}

	_, run := journal(t)
	t.Setenv("TUKI_FILE", tukiFile)
	t.Setenv("MORI_CONFIG", cfg)

	if _, err := run("today", "--from-tuki"); err == nil {
		t.Error("--from-tuki read tuki after the integration was turned off")
	}
}

// mori reads tuki's file; it never writes to it.
func TestMoriNeverWritesToTuki(t *testing.T) {
	dir := t.TempDir()
	tukiFile := filepath.Join(dir, "tasks.json")
	original := tasksToday(t)
	if err := os.WriteFile(tukiFile, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	_, run := journal(t)
	t.Setenv("TUKI_FILE", tukiFile)

	for _, args := range [][]string{
		{"today", "--from-tuki"},
		{"today", "--from-tuki", "--write"},
		{"config"},
		{"today"},
	} {
		if _, err := run(args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	after, err := os.ReadFile(tukiFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != original {
		t.Errorf("mori changed tuki's file:\n%s", after)
	}
}
