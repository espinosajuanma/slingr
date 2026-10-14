package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathForSeparatesAliasesAndHonorsOverrides(t *testing.T) {
	home := t.TempDir()
	xdg := t.TempDir()

	solutions := PathFor("solutions", home, xdg, "")
	hq := PathFor("hq", home, xdg, "")
	if solutions == hq {
		t.Fatal("expected distinct config files for executable aliases")
	}
	if got, want := solutions, filepath.Join(xdg, "solutions", "config.json"); got != want {
		t.Fatalf("XDG config path = %q, want %q", got, want)
	}
	if got, want := PathFor("solutions", home, "", ""), filepath.Join(home, ".config", "solutions", "config.json"); got != want {
		t.Fatalf("HOME config path = %q, want %q", got, want)
	}
	if got, want := PathFor("solutions", home, xdg, "/custom/config"), filepath.Join("/custom/config", "config.json"); got != want {
		t.Fatalf("CONFIG_DIR path = %q, want %q", got, want)
	}
}

func TestFileStoreRoundTripAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "solutions", "config.json")
	store := NewFileStore(path)
	values := map[string]string{
		"app":                 "my app",
		"email":               "user@example.com",
		"custom.key":          "line one\nline two",
		"solutions.dev.token": "private-token",
	}
	if err := store.Save(values); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range values {
		if got[key] != want {
			t.Errorf("config[%q] = %q, want %q", key, got[key], want)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("config permissions = %o, want 600", got)
	}
}

func TestFileStoreMissingFileReturnsEmptySettings(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))
	values, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if values == nil || len(values) != 0 {
		t.Fatalf("missing config produced %#v, want empty map", values)
	}
}
