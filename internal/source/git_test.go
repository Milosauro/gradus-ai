package source_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/source"
)

func runGitTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Gradus Test",
		"GIT_AUTHOR_EMAIL=test@example.test",
		"GIT_COMMITTER_NAME=Gradus Test",
		"GIT_COMMITTER_EMAIL=test@example.test",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func TestResolveGitSourceAndOfflinePolicy(t *testing.T) {
	// 1. Create a local bare/source git repo with tag v1.0.0
	remoteRepo := t.TempDir()
	runGitTest(t, remoteRepo, "init")
	if err := os.MkdirAll(filepath.Join(remoteRepo, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remoteRepo, "level.yaml"), []byte("schema: gradus/v1\nid: team\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remoteRepo, "rules", "TEAM-01.md"), []byte("---\nid: TEAM-01\ntitle: T\nseverity: MUST\n---\nRule.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, remoteRepo, "add", ".")
	runGitTest(t, remoteRepo, "commit", "-m", "init team rules")
	runGitTest(t, remoteRepo, "tag", "v1.0.0")

	workspace := t.TempDir()

	// 2. Resolve pinned git tag v1.0.0
	res, err := source.ResolveGitSource(workspace, "level", "team", schema.SourceRef{
		Git: remoteRepo,
		Ref: "v1.0.0",
	}, &schema.OfflinePolicy{Mode: "cache", MaxAge: "7d"}, nil)
	if err != nil {
		t.Fatalf("expected git source to resolve, got: %v", err)
	}
	if res.Commit == "" || !strings.HasPrefix(res.Hash, "sha256:") {
		t.Fatalf("expected commit and hash, got: %+v", res)
	}

	// 3. Simulate unreachable remote with fresh cache -> succeeds with fallback note
	badURL := filepath.Join(t.TempDir(), "does-not-exist.git")
	cachedRes, err := source.ResolveGitSource(workspace, "level", "team", schema.SourceRef{
		Git: badURL,
		Ref: "v1.0.0",
	}, &schema.OfflinePolicy{Mode: "cache", MaxAge: "7d"}, nil)
	if err != nil {
		t.Fatalf("expected offline cache fallback to succeed, got: %v", err)
	}
	if len(cachedRes.Notes) == 0 || !strings.Contains(cachedRes.Notes[0], "offline cache fallback") {
		t.Fatalf("expected offline cache fallback note, got: %v", cachedRes.Notes)
	}

	// 4. Simulate fail-closed when unreachable -> error
	if _, err := source.ResolveGitSource(workspace, "level", "team", schema.SourceRef{
		Git: badURL,
		Ref: "v1.0.0",
	}, &schema.OfflinePolicy{Mode: "fail-closed"}, nil); err == nil {
		t.Fatalf("expected fail-closed offline policy to fail when remote is unreachable")
	}

	// 5. Simulate expired cache -> error
	fetchedAt := filepath.Join(workspace, ".gradus", "cache", "team", ".fetched_at")
	oldTS := time.Now().Add(-10 * 24 * time.Hour).UTC().Format(time.RFC3339)
	if err := os.WriteFile(fetchedAt, []byte(oldTS+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ResolveGitSource(workspace, "level", "team", schema.SourceRef{
		Git: badURL,
		Ref: "v1.0.0",
	}, &schema.OfflinePolicy{Mode: "cache", MaxAge: "7d"}, nil); err == nil || !strings.Contains(err.Error(), "cache expired") {
		t.Fatalf("expected expired cache error, got: %v", err)
	}
}
