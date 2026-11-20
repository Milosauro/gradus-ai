package hooks_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/hooks"
)

func TestHooksManager(t *testing.T) {
	// 1. Auto-detection priority: pre-commit -> lefthook -> husky -> native
	d1 := t.TempDir()
	_ = os.WriteFile(filepath.Join(d1, ".pre-commit-config.yaml"), []byte("repos: []\n"), 0o644)
	_ = os.WriteFile(filepath.Join(d1, "lefthook.yml"), []byte(""), 0o644)
	if got := hooks.DetectManager(d1, "auto"); got != hooks.ManagerPreCommit {
		t.Fatalf("expected pre-commit priority, got %q", got)
	}

	d2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(d2, "lefthook.yml"), []byte(""), 0o644)
	if got := hooks.DetectManager(d2, "auto"); got != hooks.ManagerLefthook {
		t.Fatalf("expected lefthook, got %q", got)
	}

	d3 := t.TempDir()
	_ = os.MkdirAll(filepath.Join(d3, ".husky"), 0o755)
	if got := hooks.DetectManager(d3, "auto"); got != hooks.ManagerHusky {
		t.Fatalf("expected husky, got %q", got)
	}

	d4 := t.TempDir()
	if got := hooks.DetectManager(d4, "auto"); got != hooks.ManagerNative {
		t.Fatalf("expected native fallback, got %q", got)
	}

	// 2. Idempotent install for lefthook
	if _, err := hooks.InstallHooks(d2, hooks.ManagerLefthook, false); err != nil {
		t.Fatal(err)
	}
	if _, err := hooks.InstallHooks(d2, hooks.ManagerLefthook, false); err != nil {
		t.Fatal(err)
	}
	lhBytes, _ := os.ReadFile(filepath.Join(d2, "lefthook.yml"))
	if strings.Count(string(lhBytes), "gradus-verify") != 1 {
		t.Fatalf("expected idempotent lefthook.yml with 1 gradus-verify entry, got:\n%s", string(lhBytes))
	}

	// 3. Native requires explicit allowNative consent (D-72)
	cmd := exec.Command("git", "init")
	cmd.Dir = d4
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := hooks.InstallHooks(d4, hooks.ManagerNative, false); err == nil {
		t.Fatalf("expected native hook install without consent to fail")
	}
	if _, err := hooks.InstallHooks(d4, hooks.ManagerNative, true); err != nil {
		t.Fatalf("expected native hook install with --native consent to succeed, got: %v", err)
	}
}
