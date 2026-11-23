package source_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/source"
)

func TestVerifyGitSignature(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	tmp := t.TempDir()
	keyPath := filepath.Join(tmp, "signing_key")
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen failed: %v\n%s", err, string(out))
	}
	pubBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	allowedSigners := filepath.Join(tmp, "allowed_signers")
	if err := os.WriteFile(allowedSigners, []byte("test@gradus.local "+string(pubBytes)), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	gitRun("init")
	gitRun("config", "user.name", "Test")
	gitRun("config", "user.email", "test@gradus.local")
	gitRun("config", "gpg.format", "ssh")
	gitRun("config", "user.signingkey", keyPath)
	gitRun("config", "gpg.ssh.allowedSignersFile", allowedSigners)

	if err := os.WriteFile(filepath.Join(repo, "level.yaml"), []byte("schema: gradus/v1\nid: team\nversion: 1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", "level.yaml")
	gitRun("commit", "-m", "unsigned commit")

	// Unsigned commit must fail verification and return "none"
	sig, err := source.VerifyGitSignature(repo, "HEAD")
	if err == nil || sig != "none" {
		t.Fatalf("expected unsigned commit to fail with sig=none, got sig=%q err=%v", sig, err)
	}

	// Leading '-' ref must be rejected (NFR2)
	if sig, err := source.VerifyGitSignature(repo, "-evil"); err == nil || sig != "none" || !strings.Contains(err.Error(), "cannot start with '-'") {
		t.Fatalf("expected leading '-' ref rejection, got sig=%q err=%v", sig, err)
	}

	// Signed commit + signed annotated tag
	if err := os.WriteFile(filepath.Join(repo, "level.yaml"), []byte("schema: gradus/v1\nid: team\nversion: 1.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", "level.yaml")
	gitRun("commit", "-S", "-m", "signed commit")
	gitRun("tag", "-s", "v1.1.0", "-m", "signed release v1.1.0")

	sig, err = source.VerifyGitSignature(repo, "HEAD")
	if err != nil || sig != "verified" {
		t.Fatalf("expected signed commit to verify, got sig=%q err=%v", sig, err)
	}

	sig, err = source.VerifyGitSignature(repo, "v1.1.0")
	if err != nil || sig != "verified" {
		t.Fatalf("expected signed tag v1.1.0 to verify, got sig=%q err=%v", sig, err)
	}
}
