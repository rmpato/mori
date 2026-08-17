// Package config holds the few things mori lets you change. There is
// deliberately very little here: a journal that needs configuring before it
// will take a sentence is a journal you stop writing in.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Tuki is how mori relates to its sibling.
//
// The default is that mori reads tuki's tasks if tuki is installed, and
// otherwise never mentions it. Writing to tuki is not a setting: mori reads
// from tuki, and mori does not control tuki.
type Tuki struct {
	// Enabled is nil when you haven't said, which means "if tuki is there".
	Enabled *bool `json:"enabled,omitempty"`
	// File overrides where tuki's tasks live. Empty means tuki's own default.
	File string `json:"file,omitempty"`
}

// Config is the whole file.
type Config struct {
	Tuki Tuki `json:"tuki"`

	// extra keeps anything a future mori writes here, so an older binary
	// reading a newer file and saving it doesn't throw the rest away.
	extra map[string]json.RawMessage
}

// Default is mori with nothing configured, which is how it is meant to be
// used.
func Default() Config { return Config{} }

// Path is where the config lives, without touching the disk.
func Path() (string, error) {
	if p := os.Getenv("MORI_CONFIG"); p != "" {
		return p, nil
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "mori", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("mori can't find your home directory: %w", err)
	}
	return filepath.Join(home, ".config", "mori", "config.json"), nil
}

// Load reads the config. A file that isn't there is not an error — it is the
// normal case.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("reading %s: %w", path, err)
	}

	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Default(), fmt.Errorf("%s isn't valid JSON: %w", path, err)
	}
	if err := json.Unmarshal(raw, &c.extra); err != nil {
		return Default(), fmt.Errorf("%s isn't valid JSON: %w", path, err)
	}
	delete(c.extra, "tuki")
	return c, nil
}

// TukiEnabled reports whether mori should read tuki, given whether tuki's file
// is actually there. Saying nothing means "if it's there"; saying so either
// way is respected.
func (c Config) TukiEnabled(present bool) bool {
	if c.Tuki.Enabled != nil {
		return *c.Tuki.Enabled
	}
	return present
}
