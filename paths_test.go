package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDataRootPrefersEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NARABERU_DATA_DIR", dir)
	got := resolveDataRoot()
	if filepath.Clean(got) != filepath.Clean(dir) {
		t.Fatalf("resolveDataRoot=%q want %q", got, dir)
	}
}

func TestResolveDataRootDefaultIsDataUnderExeOrCwd(t *testing.T) {
	t.Setenv("NARABERU_DATA_DIR", "")
	got := resolveDataRoot()
	if filepath.Base(got) != "data" {
		t.Fatalf("expected …/data, got %q", got)
	}
}

func TestPathInsideDir(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "thumbnails", "a.jpg")
	if !pathInsideDir(dir, inside) {
		t.Fatal("expected inside")
	}
	if pathInsideDir(dir, filepath.Join(dir, "..", "other", "x")) {
		t.Fatal("expected outside")
	}
}

func TestEnsureDataLayout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	if err := ensureDataLayout(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"thumbnails", "backups", "sounds", "webview"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}
