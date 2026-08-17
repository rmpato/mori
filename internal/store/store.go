// Package store keeps mori's journal as ordinary Markdown files, one per day,
// in year/month directories. No database, no index, no single file that can
// take the whole journal down with it — just pages you could read, edit, back
// up, or move somewhere else without mori's help.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rmpato/mori/internal/entry"
)

// File permissions. A journal is nobody else's business, and the default
// umask is not careful enough for it.
const (
	dirPerm  fs.FileMode = 0o700
	filePerm fs.FileMode = 0o600
)

// ErrStop tells Walk to stop early without treating it as a failure.
var ErrStop = errors.New("stop walking")

// Journal is the seam between mori and where its pages live. Today there is
// one implementation, writing Markdown files; if search ever outgrows a plain
// scan, an indexed store slots in here without the files changing shape.
type Journal interface {
	Get(d entry.Date) (entry.Entry, error)
	Put(e entry.Entry) error
	Has(d entry.Date) (bool, error)
	Dates(from, to entry.Date) ([]entry.Date, error)
	Walk(from, to entry.Date, fn func(entry.Entry) error) error
}

var _ Journal = (*Store)(nil)

// Store is a journal kept in a directory tree.
type Store struct {
	root string
}

// New returns a Store rooted at an explicit directory.
func New(root string) *Store { return &Store{root: root} }

// Default returns a Store at mori's usual home.
func Default() (*Store, error) {
	root, err := DefaultRoot()
	if err != nil {
		return nil, err
	}
	return New(root), nil
}

// DefaultRoot resolves where the journal lives without touching the disk,
// honouring MORI_DIR, MORI_HOME, and XDG_DATA_HOME in that order.
func DefaultRoot() (string, error) {
	if p := os.Getenv("MORI_DIR"); p != "" {
		return p, nil
	}
	if dir := os.Getenv("MORI_HOME"); dir != "" {
		return filepath.Join(dir, "journal"), nil
	}
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "mori", "journal"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("mori can't find your home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "mori", "journal"), nil
}

// Root is the directory this store reads and writes.
func (s *Store) Root() string { return s.root }

// Path is where a given day is kept: 2026/08/2026-08-17.md. The filename
// repeats the directories on purpose, so a page still says what it is if you
// ever drag it somewhere else.
func (s *Store) Path(d entry.Date) string {
	return filepath.Join(s.root,
		fmt.Sprintf("%04d", d.Year),
		fmt.Sprintf("%02d", int(d.Month)),
		d.String()+".md")
}

// Get reads a day.
//
// A day you never wrote is not an error: it comes back as a blank page, which
// is the honest answer to "what did I write on the third of May".
func (s *Store) Get(d entry.Date) (entry.Entry, error) {
	raw, err := os.ReadFile(s.Path(d))
	if errors.Is(err, fs.ErrNotExist) {
		return entry.New(d), nil
	}
	if err != nil {
		return entry.Entry{}, fmt.Errorf("reading %s: %w", d, err)
	}
	return entry.Parse(d, raw), nil
}

// Has reports whether a day has a page on disk.
func (s *Store) Has(d entry.Date) (bool, error) {
	_, err := os.Stat(s.Path(d))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking %s: %w", d, err)
	}
	return true, nil
}

// LastWritten is when a day's file was last touched, and whether it exists at
// all. It answers one question — have you been away from this page long
// enough that coming back to it counts as a new sitting — so a filesystem
// timestamp that a copy or a checkout might have disturbed is good enough.
func (s *Store) LastWritten(d entry.Date) (time.Time, bool, error) {
	fi, err := os.Stat(s.Path(d))
	if errors.Is(err, fs.ErrNotExist) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("checking %s: %w", d, err)
	}
	return fi.ModTime(), true, nil
}

// Put writes a day.
//
// Emptying a page removes the file rather than leaving a husk behind, so the
// calendar never fills up with days you opened and didn't write in.
func (s *Store) Put(e entry.Entry) error {
	if e.Date.IsZero() {
		return errors.New("mori won't save a page without a date")
	}
	path := s.Path(e.Date)

	if e.IsEmpty() && e.Mood == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("removing %s: %w", e.Date, err)
		}
		s.prune(filepath.Dir(path))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	return writeFileAtomic(path, e.Format())
}

// prune removes a month directory, and then a year directory, once the last
// page in them is gone. Failures are ignored: an empty directory is untidy,
// never wrong.
func (s *Store) prune(dir string) {
	for range 2 {
		if dir == s.root || !strings.HasPrefix(dir, s.root) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// Dates lists the days that actually have pages, oldest first.
//
// A zero from or to means "as far back as there is" and "up to the last day
// written", so Dates(entry.Date{}, entry.Date{}) is the whole journal.
func (s *Store) Dates(from, to entry.Date) ([]entry.Date, error) {
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		from, to = to, from
	}

	years, err := numericDirs(s.root, 4)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil // a journal you haven't started is not a problem
	}
	if err != nil {
		return nil, err
	}

	var out []entry.Date
	for _, y := range years {
		if !from.IsZero() && y < from.Year {
			continue
		}
		if !to.IsZero() && y > to.Year {
			continue
		}
		yearDir := filepath.Join(s.root, fmt.Sprintf("%04d", y))
		months, err := numericDirs(yearDir, 2)
		if err != nil {
			return nil, err
		}
		for _, m := range months {
			if m < 1 || m > 12 {
				continue
			}
			dates, err := monthDates(filepath.Join(yearDir, fmt.Sprintf("%02d", m)), y, m)
			if err != nil {
				return nil, err
			}
			for _, d := range dates {
				if !from.IsZero() && d.Before(from) {
					continue
				}
				if !to.IsZero() && d.After(to) {
					continue
				}
				out = append(out, d)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out, nil
}

// Walk reads the days in a range, newest first, which is the order you look
// back in. Returning ErrStop from fn ends the walk quietly.
func (s *Store) Walk(from, to entry.Date, fn func(entry.Entry) error) error {
	dates, err := s.Dates(from, to)
	if err != nil {
		return err
	}
	for i := len(dates) - 1; i >= 0; i-- {
		e, err := s.Get(dates[i])
		if err != nil {
			return err
		}
		if err := fn(e); err != nil {
			if errors.Is(err, ErrStop) {
				return nil
			}
			return err
		}
	}
	return nil
}

// numericDirs lists the subdirectories whose names are exactly n digits,
// sorted ascending. Anything else in the tree is left alone.
func numericDirs(dir string, n int) ([]int, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var out []int
	for _, e := range ents {
		if !e.IsDir() || len(e.Name()) != n {
			continue
		}
		v, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	sort.Ints(out)
	return out, nil
}

// monthDates lists the pages in one month directory. A file whose name
// doesn't match the day it sits under is ignored rather than trusted.
func monthDates(dir string, year, month int) ([]entry.Date, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var out []entry.Date
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name, ok := strings.CutSuffix(e.Name(), ".md")
		if !ok {
			continue
		}
		d, err := parseFilename(name)
		if err != nil || d.Year != year || int(d.Month) != month {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

// parseFilename reads a strict "2026-08-17" name, and only that: this is a
// filename, not something a person typed, so the friendly date vocabulary
// would only let nonsense through.
func parseFilename(name string) (entry.Date, error) {
	var y, m, d int
	if _, err := fmt.Sscanf(name, "%4d-%2d-%2d", &y, &m, &d); err != nil {
		return entry.Date{}, err
	}
	if len(name) != 10 || m < 1 || m > 12 || d < 1 || d > 31 {
		return entry.Date{}, fmt.Errorf("%q is not a date", name)
	}
	return entry.Date{Year: y, Month: time.Month(m), Day: d}, nil
}
