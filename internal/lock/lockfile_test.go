package lock_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/lock"
)

func TestLockfileReadWriteAndVerify(t *testing.T) {
	tmp := t.TempDir()
	lockPath := filepath.Join(tmp, "gradus.lock")

	lf := lock.Lockfile{
		DefinitionHash: lock.HashBytes([]byte("schema: gradus/v1\n")),
		Entries: []lock.Entry{
			{
				Kind:      "level",
				ID:        "team",
				Source:    "git@github.com:acme/rules.git",
				Ref:       "v1.2.0",
				Commit:    "4f2a9c1",
				Hash:      "sha256:" + strings.Repeat("a", 64),
				Signature: "none",
			},
			{
				Kind:   "level",
				ID:     "project",
				Source: ".gradus/levels/project",
				Hash:   "sha256:" + strings.Repeat("b", 64),
			},
			{
				Kind:   "level",
				ID:     "user",
				Source: "~/.gradus/levels/user",
				Hash:   "sha256:" + strings.Repeat("c", 64),
			},
		},
		ViewHash: "sha256:" + strings.Repeat("d", 64),
	}

	if err := lock.WriteLockfile(lockPath, lf); err != nil {
		t.Fatalf("failed to write lockfile: %v", err)
	}

	readBack, err := lock.ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}

	// User level must be excluded (§9.6)
	if _, ok := readBack.FindEntry("level", "user"); ok {
		t.Fatalf("expected user level to be excluded from gradus.lock")
	}
	if len(readBack.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(readBack.Entries))
	}

	teamEntry, ok := readBack.FindEntry("level", "team")
	if !ok {
		t.Fatalf("expected team entry in lockfile")
	}
	if err := lock.VerifyLockedEntry(teamEntry, teamEntry.Hash); err != nil {
		t.Fatalf("expected matching hash to verify, got: %v", err)
	}
	if err := lock.VerifyLockedEntry(teamEntry, "sha256:"+strings.Repeat("f", 64)); err == nil || !strings.Contains(err.Error(), "lock integrity mismatch") {
		t.Fatalf("expected lock integrity mismatch error, got: %v", err)
	}
}
