package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Not having a config file is the normal case, not a problem.
func TestLoadMissingFile(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nothing.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Tuki.Enabled != nil || c.Tuki.File != "" {
		t.Errorf("Config = %+v, want the defaults", c)
	}
}

func TestLoad(t *testing.T) {
	c, err := Load(write(t, `{"tuki": {"enabled": false, "file": "/somewhere/tasks.json"}}`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Tuki.Enabled == nil || *c.Tuki.Enabled {
		t.Errorf("Enabled = %v, want an explicit false", c.Tuki.Enabled)
	}
	if c.Tuki.File != "/somewhere/tasks.json" {
		t.Errorf("File = %q", c.Tuki.File)
	}
}

func TestLoadRejectsNonsense(t *testing.T) {
	if _, err := Load(write(t, "not json")); err == nil {
		t.Error("Load accepted a file that isn't JSON")
	}
}

// Saying nothing means "if tuki is there"; saying so either way is respected.
func TestTukiEnabled(t *testing.T) {
	yes, no := true, false

	tests := []struct {
		name    string
		enabled *bool
		present bool
		want    bool
	}{
		{"nothing said, tuki installed", nil, true, true},
		{"nothing said, no tuki", nil, false, false},
		{"turned off, tuki installed", &no, true, false},
		{"turned on, no tuki", &yes, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{Tuki: Tuki{Enabled: tt.enabled}}
			if got := c.TukiEnabled(tt.present); got != tt.want {
				t.Errorf("TukiEnabled(%v) = %v, want %v", tt.present, got, tt.want)
			}
		})
	}
}

// An older mori reading a newer file must not quietly throw the rest away.
func TestUnknownSettingsAreKept(t *testing.T) {
	c, err := Load(write(t, `{"tuki": {"enabled": true}, "theme": {"accent": "moss"}}`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	raw, ok := c.extra["theme"]
	if !ok {
		t.Fatalf("the unknown setting was dropped: %+v", c.extra)
	}
	var theme map[string]string
	if err := json.Unmarshal(raw, &theme); err != nil {
		t.Fatal(err)
	}
	if theme["accent"] != "moss" {
		t.Errorf("theme = %v", theme)
	}
	if _, ok := c.extra["tuki"]; ok {
		t.Error("a setting mori does understand was also kept as unknown")
	}
}

func TestPath(t *testing.T) {
	t.Setenv("MORI_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	t.Run("MORI_CONFIG wins", func(t *testing.T) {
		t.Setenv("MORI_CONFIG", "/somewhere/mori.json")
		if got, _ := Path(); got != "/somewhere/mori.json" {
			t.Errorf("Path = %q", got)
		}
	})
	t.Run("then XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		want := filepath.Join("/xdg", "mori", "config.json")
		if got, _ := Path(); got != want {
			t.Errorf("Path = %q, want %q", got, want)
		}
	})
	t.Run("otherwise the usual place", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".config", "mori", "config.json")
		if got, _ := Path(); got != want {
			t.Errorf("Path = %q, want %q", got, want)
		}
	})
}
