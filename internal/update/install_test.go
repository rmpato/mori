package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// install.sh and AssetName both build the release archive's name from
// scratch, and if they ever stop agreeing, one of the two ways of installing
// mori quietly breaks. This runs the real script with a stub curl on the PATH
// and checks what it tried to download.
func TestInstallScriptAsksForTheSameArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a POSIX shell script")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Skipf("no install.sh: %v", err)
	}

	dir := t.TempDir()
	log := filepath.Join(dir, "requested")

	// A curl that records what it was asked for and then fails, so the script
	// stops before it can touch anything real. wget is shadowed too, so the
	// fallback can't quietly take over.
	stub := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in http*) echo \"$a\" >>" + log + ";; esac; done\nexit 1\n"
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"curl", "wget"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(stub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("sh", script,
		"--version", "v0.2.0",
		"--dir", filepath.Join(dir, "install"),
		"--no-modify-path")
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"NO_COLOR=1",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("the script succeeded with a stub curl:\n%s", out)
	}

	requested, readErr := os.ReadFile(log)
	if readErr != nil {
		t.Fatalf("the script downloaded nothing:\n%s", out)
	}

	want := AssetName("v0.2.0", runtime.GOOS, runtime.GOARCH)
	if !strings.Contains(string(requested), want) {
		t.Errorf("install.sh asked for:\n%s\nbut AssetName says %s", requested, want)
	}
	// And it must be from the right repository.
	if !strings.Contains(string(requested), "rmpato/mori") {
		t.Errorf("install.sh downloaded from somewhere unexpected:\n%s", requested)
	}
}

// The installer must not need the network to explain itself.
func TestInstallScriptHelp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a POSIX shell script")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Skipf("no install.sh: %v", err)
	}

	out, err := exec.Command("sh", script, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, out)
	}
	for _, want := range []string{"--dir", "--version", "--no-modify-path"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("--help doesn't mention %s:\n%s", want, out)
		}
	}
}
