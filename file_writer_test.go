package asmgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiskWriterRoundTrip(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "out.s")
	content := []byte("line1\nline2\n")

	if err := NewDiskWriter().WriteFile(target, content); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch: %q", got)
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %v, want 0600", fi.Mode().Perm())
	}

	if m, err := compareWithExisting(target, content); err != nil || m != nil {
		t.Fatalf("expected match, got mismatch=%+v err=%v", m, err)
	}

	if m, _ := compareWithExisting(target, []byte("line1\nDIFF\n")); m == nil || m.Line != 2 {
		t.Fatalf("expected mismatch on line 2, got %+v", m)
	}

	if m, err := compareWithExisting(filepath.Join(dir, "nope.s"), content); err != nil || m == nil || m.Expected != "<file does not exist>" {
		t.Fatalf("expected missing-file mismatch, got m=%+v err=%v", m, err)
	}

	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 || entries[0].Name() != "out.s" {
		t.Fatalf("stray files left: %v", entries)
	}
}
