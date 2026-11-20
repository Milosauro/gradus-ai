package lock_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/lock"
)

func TestHashDirectoryAndBytes(t *testing.T) {
	// CRLF vs LF bytes must hash identically
	lf := lock.HashBytes([]byte("line1\nline2\n"))
	crlf := lock.HashBytes([]byte("line1\r\nline2\r\n"))
	if lf != crlf {
		t.Fatalf("expected CRLF and LF hashes to match: %s vs %s", lf, crlf)
	}
	if !strings.HasPrefix(lf, "sha256:") || len(lf) != len("sha256:")+64 {
		t.Fatalf("unexpected hash format: %q", lf)
	}

	// Two directories with identical content (one CRLF, one LF, plus ignored .git/.fetched_at)
	dirLF := t.TempDir()
	dirCRLF := t.TempDir()

	for _, d := range []struct {
		dir  string
		eol  string
		meta bool
	}{
		{dir: dirLF, eol: "\n", meta: false},
		{dir: dirCRLF, eol: "\r\n", meta: true},
	} {
		if err := os.MkdirAll(filepath.Join(d.dir, "rules"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d.dir, "level.yaml"), []byte("schema: gradus/v1"+d.eol+"id: team"+d.eol), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d.dir, "rules", "TEAM-01.md"), []byte("---"+d.eol+"id: TEAM-01"+d.eol+"---"+d.eol+"Rule."+d.eol), 0o644); err != nil {
			t.Fatal(err)
		}
		if d.meta {
			if err := os.MkdirAll(filepath.Join(d.dir, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d.dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d.dir, ".fetched_at"), []byte("2026-11-19T00:00:00Z\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	h1, err := lock.HashDirectory(dirLF)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := lock.HashDirectory(dirCRLF)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("expected identical directory hashes across CRLF/LF and ignored metadata: %s vs %s", h1, h2)
	}
}
