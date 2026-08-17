package facts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rmpato/mori/internal/entry"
)

// mori reads tuki's file directly rather than importing tuki.
//
// Importing it would lock the two projects' versions together and make mori's
// build depend on tuki's refactors — exactly the coupling that would turn two
// small tools into one big one. What is declared below is only the handful of
// fields mori needs; everything else in the file is ignored and left alone.
//
// The file is atomically written JSON with a version in it, which is a
// perfectly good contract between two programs that agree not to reach any
// further into each other than this.

// tukiVersion is the newest on-disk schema this understands.
const tukiVersion = 1

// ErrUnsupported means tuki's file is newer than this mori knows how to read.
// Callers should be quiet about it rather than alarming: the journal is fine,
// only the extra context is missing.
var ErrUnsupported = errors.New("this tuki file is newer than mori understands")

type tukiFile struct {
	Version int        `json:"version"`
	Tasks   []tukiTask `json:"tasks"`
}

type tukiTask struct {
	Text      string     `json:"text"`
	Tag       string     `json:"tag"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"created_at"`
	DoneAt    *time.Time `json:"done_at"`
	Due       *time.Time `json:"due"`
}

// tuki is a Source backed by tuki's task file.
//
// It holds on to what it last read, and re-reads only when the file has
// actually changed, because asking about a month means asking about thirty
// days and there is no sense parsing the same JSON thirty times. Keying the
// cache on the modification time means a task ticked off in tuki while mori
// is open still shows up the next time you look.
type tuki struct {
	path string

	mu     sync.Mutex
	loaded bool
	mod    time.Time
	size   int64
	file   tukiFile
}

// Tuki returns a read-only view of tuki's tasks.
//
// mori never writes to this file. Not by default, and not behind a flag: mori
// reads from tuki, and mori does not control tuki.
func Tuki(path string) Source { return &tuki{path: path} }

// TukiPath resolves where tuki keeps its file, using tuki's own rules so the
// two agree without either having to ask the other.
func TukiPath() (string, error) {
	if p := os.Getenv("TUKI_FILE"); p != "" {
		return p, nil
	}
	if dir := os.Getenv("TUKI_HOME"); dir != "" {
		return filepath.Join(dir, "tasks.json"), nil
	}
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "tuki", "tasks.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("mori can't find your home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "tuki", "tasks.json"), nil
}

// Available reports whether there is a tuki file to read at all. A missing one
// is not a problem and not worth mentioning: mori simply doesn't offer the
// context.
func Available(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Day is what tuki knows about a day.
//
// A note on the past: tuki records when a task was finished, but not what its
// state was on some earlier evening. So for today this is exact, and for a day
// last March it is the best reading available from what tuki knows now —
// which is the right trade for something being offered as context beside a
// blank page.
func (t *tuki) Day(d entry.Date) (Snapshot, error) {
	file, err := t.load()
	if err != nil {
		return Snapshot{}, err
	}

	s := Snapshot{Source: "tuki"}
	for _, task := range file.Tasks {
		if task.Text == "" {
			continue
		}
		item := Item{Text: task.Text, Tag: entry.NormalizeTag(task.Tag)}

		switch {
		case task.Done:
			if task.DoneAt != nil && entry.DateOf(*task.DoneAt) == d {
				s.Done = append(s.Done, item)
			}
		case openOn(task, d):
			s.Todo = append(s.Todo, item)
		}
	}
	return s, nil
}

// load reads tuki's file, or hands back what was read last time if nothing
// about it has changed since.
func (t *tuki) load() (tukiFile, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	fi, err := os.Stat(t.path)
	if errors.Is(err, fs.ErrNotExist) {
		// No tuki, no context, no complaint.
		return tukiFile{}, nil
	}
	if err != nil {
		return tukiFile{}, fmt.Errorf("reading tuki's tasks: %w", err)
	}
	if t.loaded && fi.ModTime().Equal(t.mod) && fi.Size() == t.size {
		return t.file, nil
	}

	raw, err := os.ReadFile(t.path)
	if errors.Is(err, fs.ErrNotExist) {
		return tukiFile{}, nil
	}
	if err != nil {
		return tukiFile{}, fmt.Errorf("reading tuki's tasks: %w", err)
	}

	var file tukiFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return tukiFile{}, fmt.Errorf("%s doesn't look like a tuki file: %w", t.path, err)
	}
	if file.Version > tukiVersion {
		return tukiFile{}, fmt.Errorf("%w (%s is version %d)", ErrUnsupported, t.path, file.Version)
	}

	t.file, t.loaded, t.mod, t.size = file, true, fi.ModTime(), fi.Size()
	return file, nil
}

// openOn reports whether an unfinished task was on your plate that day.
//
// That means it existed by then, and either it was due by then or you made it
// that day. Without the second half, every task you have ever left open would
// follow you backwards through the whole journal.
func openOn(task tukiTask, d entry.Date) bool {
	created := entry.DateOf(task.CreatedAt)
	if !task.CreatedAt.IsZero() && created.After(d) {
		return false
	}
	if task.Due != nil {
		return !entry.DateOf(*task.Due).After(d)
	}
	return created == d
}
