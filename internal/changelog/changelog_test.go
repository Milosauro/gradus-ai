package changelog_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/changelog"
)

func TestDiffLevelChangelog(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "team-level")
	if err := os.MkdirAll(filepath.Join(repo, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
		return strings.TrimSpace(string(out))
	}

	gitRun("init")
	gitRun("config", "user.name", "Test")
	gitRun("config", "user.email", "test@gradus.local")

	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// v1.0.0
	write("level.yaml", "schema: gradus/v1\nid: team\nversion: 1.0.0\n")
	write("rules/TEAM-001.md", "---\nid: TEAM-001\ntitle: Keep tests fast\nseverity: SHOULD\nenforcement: review\n---\n\nKeep unit tests under 5s.\n")
	write("rules/TEAM-002.md", "---\nid: TEAM-002\ntitle: Old rule\nseverity: MAY\nenforcement: prompt\n---\n\nTo be removed.\n")
	gitRun("add", ".")
	gitRun("commit", "-m", "v1.0.0")
	gitRun("tag", "v1.0.0")

	// v1.1.0: modify TEAM-001, remove TEAM-002, add TEAM-003
	write("level.yaml", "schema: gradus/v1\nid: team\nversion: 1.1.0\nmin_version: 1.1.0\n")
	write("rules/TEAM-001.md", "---\nid: TEAM-001\ntitle: Keep tests fast\nseverity: MUST\nenforcement: check\nchecker:\n  name: pattern\n  pattern: time\\.Sleep\n---\n\nKeep unit tests under 2s.\n")
	_ = os.Remove(filepath.Join(repo, "rules", "TEAM-002.md"))
	write("rules/TEAM-003.md", "---\nid: TEAM-003\ntitle: New security rule\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: \"echo secret\"\n  decision: deny\n---\n\nBlock hardcoded tokens.\n")
	gitRun("add", "-A")
	gitRun("commit", "-m", "v1.1.0")
	gitRun("tag", "v1.1.0")

	// Reject leading '-' refs (NFR2)
	if _, err := changelog.DiffLevel(repo, "team", "-evil", "v1.1.0"); err == nil {
		t.Fatalf("expected leading '-' ref rejection")
	}

	rep, err := changelog.DiffLevel(repo, "team", "v1.0.0", "v1.1.0")
	if err != nil {
		t.Fatalf("DiffLevel failed: %v", err)
	}

	if len(rep.Added) != 1 || rep.Added[0].ID != "TEAM-003" {
		t.Fatalf("expected TEAM-003 added, got: %+v", rep.Added)
	}
	if len(rep.Removed) != 1 || rep.Removed[0].ID != "TEAM-002" {
		t.Fatalf("expected TEAM-002 removed, got: %+v", rep.Removed)
	}
	if len(rep.Modified) != 1 || rep.Modified[0].ID != "TEAM-001" {
		t.Fatalf("expected TEAM-001 modified, got: %+v", rep.Modified)
	}
	if len(rep.MetadataChanges) < 2 {
		t.Fatalf("expected version and min_version metadata changes, got: %+v", rep.MetadataChanges)
	}

	md := changelog.FormatMarkdown(rep)
	if !strings.Contains(md, "TEAM-001") || !strings.Contains(md, "TEAM-002") || !strings.Contains(md, "TEAM-003") {
		t.Fatalf("unexpected Markdown output:\n%s", md)
	}

	// Non-destructive check: git status in repo must remain clean
	if st := gitRun("status", "--short"); st != "" {
		t.Fatalf("expected clean git status after DiffLevel, got:\n%s", st)
	}
}
