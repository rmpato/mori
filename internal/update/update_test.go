package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.0.0", "v1.1.0", -1},
		{"v1.9.0", "v2.0.0", -1},
		{"v1.2", "v1.2.0", 0},

		// A pre-release comes before the version it leads to.
		{"v1.0.0-rc1", "v1.0.0", -1},
		{"v1.0.0", "v1.0.0-rc1", 1},
		{"v1.0.0-rc1", "v1.0.0-rc2", -1},

		// A development build is older than anything that was ever released.
		{"dev", "v0.1.0", -1},
		{"", "v0.1.0", -1},
		{"dev", "also nonsense", 0},
	}

	for _, tt := range tests {
		if got := Compare(tt.a, tt.b); got != tt.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	if !IsNewer("v0.1.0", "v0.2.0") {
		t.Error("v0.2.0 should be newer than v0.1.0")
	}
	if IsNewer("v0.2.0", "v0.1.0") {
		t.Error("v0.1.0 is not newer than v0.2.0")
	}
	if IsNewer("v0.2.0", "v0.2.0") {
		t.Error("a version is not newer than itself")
	}
	// Someone running a build from source should still be offered a release.
	if !IsNewer("dev", "v0.1.0") {
		t.Error("a dev build should be offered a real release")
	}
}

// The asset name has to agree with the name_template in .goreleaser.yaml, or
// nothing can find anything.
func TestAssetName(t *testing.T) {
	tests := map[string]string{
		"darwin/arm64":  "mori_0.2.0_darwin_arm64.tar.gz",
		"linux/amd64":   "mori_0.2.0_linux_amd64.tar.gz",
		"windows/amd64": "mori_0.2.0_windows_amd64.zip",
	}
	for platform, want := range tests {
		goos, goarch, _ := strings.Cut(platform, "/")
		if got := AssetName("v0.2.0", goos, goarch); got != want {
			t.Errorf("AssetName(%s) = %q, want %q", platform, got, want)
		}
		// The leading v is optional on the way in and never appears on the
		// way out.
		if got := AssetName("0.2.0", goos, goarch); got != want {
			t.Errorf("AssetName without the v = %q, want %q", got, want)
		}
	}
}

// tarGz builds a release archive holding one file.
func tarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{
		Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// releaseServer stands in for GitHub: one release, one archive, one checksums
// file, all served from memory.
func releaseServer(t *testing.T, version string, archive []byte, checksums string) *httptest.Server {
	t.Helper()

	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{
			"tag_name": %q,
			"html_url": "https://example.invalid/release",
			"body": "## Changelog\n\n* abc1234: Did a thing (@someone)",
			"assets": [
				{"name": %q, "browser_download_url": %q},
				{"name": "checksums.txt", "browser_download_url": %q}
			]
		}`, version,
			AssetName(version, runtime.GOOS, runtime.GOARCH), srv.URL+"/archive",
			srv.URL+"/checksums")
	})
	mux.HandleFunc("/archive", func(w http.ResponseWriter, _ *http.Request) {
		w.Write(archive)
	})
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, checksums)
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestLatest(t *testing.T) {
	srv := releaseServer(t, "v0.2.0", nil, "")
	c := &Client{API: srv.URL}

	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if rel.Version != "v0.2.0" {
		t.Errorf("Version = %q", rel.Version)
	}
	if !strings.Contains(rel.Notes, "Did a thing") {
		t.Errorf("Notes = %q", rel.Notes)
	}
}

func TestBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tar.gz path")
	}
	want := []byte("#!/bin/sh\necho mori\n")
	archive := tarGz(t, "mori", want)
	name := AssetName("v0.2.0", runtime.GOOS, runtime.GOARCH)

	srv := releaseServer(t, "v0.2.0", archive, sum(archive)+"  "+name+"\n")
	c := &Client{API: srv.URL}

	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Binary(context.Background(), rel, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("Binary: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A download that doesn't match its checksum is never unpacked. This is the
// whole reason the checksum file is fetched at all.
func TestBinaryRefusesAWrongChecksum(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tar.gz path")
	}
	archive := tarGz(t, "mori", []byte("not what was published"))
	name := AssetName("v0.2.0", runtime.GOOS, runtime.GOARCH)

	srv := releaseServer(t, "v0.2.0", archive, sum([]byte("something else"))+"  "+name+"\n")
	c := &Client{API: srv.URL}

	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Binary(context.Background(), rel, runtime.GOOS, runtime.GOARCH); err == nil {
		t.Fatal("Binary accepted an archive that didn't match its checksum")
	} else if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("error = %v, want it to say why", err)
	}
}

// A release with no checksums file can't be verified, so it isn't installed.
func TestBinaryRefusesAReleaseWithNoChecksums(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"tag_name": "v0.2.0", "assets": [{"name": %q, "browser_download_url": "http://example.invalid"}]}`,
			AssetName("v0.2.0", runtime.GOOS, runtime.GOARCH))
	}))
	defer srv.Close()

	c := &Client{API: srv.URL}
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Binary(context.Background(), rel, runtime.GOOS, runtime.GOARCH); err == nil {
		t.Error("Binary installed something it couldn't verify")
	}
}

func TestBinaryRefusesAPlatformWithNoBuild(t *testing.T) {
	srv := releaseServer(t, "v0.2.0", nil, "")
	c := &Client{API: srv.URL}

	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Binary(context.Background(), rel, "plan9", "mips"); err == nil {
		t.Error("Binary found a build for plan9/mips")
	}
}

func TestChecksumFor(t *testing.T) {
	sums := []byte("aaa  mori_0.2.0_linux_amd64.tar.gz\nbbb *mori_0.2.0_darwin_arm64.tar.gz\n")

	if got, ok := checksumFor(sums, "mori_0.2.0_linux_amd64.tar.gz"); !ok || got != "aaa" {
		t.Errorf("checksumFor = %q, %v", got, ok)
	}
	// GoReleaser writes "*name" in binary mode; the star isn't part of it.
	if got, ok := checksumFor(sums, "mori_0.2.0_darwin_arm64.tar.gz"); !ok || got != "bbb" {
		t.Errorf("checksumFor with a star = %q, %v", got, ok)
	}
	if _, ok := checksumFor(sums, "mori_0.2.0_windows_amd64.zip"); ok {
		t.Error("checksumFor found a file that isn't listed")
	}
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mori")
	if err := os.WriteFile(path, []byte("the old one"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Replace(path, []byte("the new one")); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "the new one" {
		t.Errorf("file = %q", got)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %o, want it executable", fi.Mode().Perm())
	}

	// The binary that was replaced doesn't stay behind on unix.
	if _, err := os.Stat(path + ".old"); runtime.GOOS != "windows" && err == nil {
		t.Error("the old binary was left lying around")
	}
}

// Better to leave the working binary in place than to install nothing.
func TestReplaceRefusesAnEmptyBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mori")
	if err := os.WriteFile(path, []byte("the old one"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Replace(path, nil); err == nil {
		t.Fatal("Replace installed an empty binary")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "the old one" {
		t.Errorf("the working binary was disturbed: %q", got)
	}
}

func TestExtractRejectsAnArchiveWithoutMori(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tar.gz path")
	}
	archive := tarGz(t, "README.md", []byte("hello"))
	if _, err := extract(archive, "linux"); err == nil {
		t.Error("extract found a binary that isn't there")
	}
}
