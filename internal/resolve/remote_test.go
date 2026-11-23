package resolve_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestResolveRemoteSignaturesAndMinVersion(t *testing.T) {
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

	globalGitCfg := filepath.Join(tmp, "gitconfig")
	if err := os.WriteFile(globalGitCfg, []byte(fmt.Sprintf("[gpg]\n\tformat = ssh\n[gpg \"ssh\"]\n\tallowedSignersFile = %s\n", filepath.ToSlash(allowedSigners))), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", globalGitCfg)

	remoteRepo := filepath.Join(tmp, "remote-team")
	if err := os.MkdirAll(filepath.Join(remoteRepo, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = remoteRepo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}
	gitRun("init")
	gitRun("config", "user.name", "Test")
	gitRun("config", "user.email", "test@gradus.local")
	gitRun("config", "gpg.format", "ssh")
	gitRun("config", "user.signingkey", keyPath)

	ruleMD := `---
id: TEAM-001
title: Team rule
severity: MUST
enforcement: prompt
---

Follow team conventions.
`
	if err := os.WriteFile(filepath.Join(remoteRepo, "rules", "TEAM-001.md"), []byte(ruleMD), 0o644); err != nil {
		t.Fatal(err)
	}

	// v1.0.0: unsigned commit + unsigned tag
	if err := os.WriteFile(filepath.Join(remoteRepo, "level.yaml"), []byte("schema: gradus/v1\nid: team\nversion: 1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", ".")
	gitRun("commit", "-m", "v1.0.0 unsigned")
	gitRun("tag", "v1.0.0")

	// v1.2.0: signed commit + signed annotated tag
	if err := os.WriteFile(filepath.Join(remoteRepo, "level.yaml"), []byte("schema: gradus/v1\nid: team\nversion: 1.2.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", "level.yaml")
	gitRun("commit", "-S", "-m", "v1.2.0 signed")
	gitRun("tag", "-s", "v1.2.0", "-m", "release v1.2.0")

	projectDir := filepath.Join(tmp, "consumer")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	remoteGit := filepath.ToSlash(remoteRepo)

	// 1. Optional signature on unsigned tag v1.0.0 -> resolves with signature: none
	cfgOptional := fmt.Sprintf(`schema: gradus/v1
levels:
  - id: team
    source: { git: %s, ref: v1.0.0 }
    signature: optional
`, remoteGit)
	if err := os.WriteFile(filepath.Join(projectDir, "gradus.yaml"), []byte(cfgOptional), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := resolve.Resolve(projectDir, resolve.Options{})
	if err != nil {
		t.Fatalf("expected optional signature on unsigned ref to pass, got: %v", err)
	}
	if len(res.Lockfile.Entries) != 1 || res.Lockfile.Entries[0].Signature != "none" {
		t.Fatalf("expected lockfile signature=none, got: %+v", res.Lockfile.Entries)
	}

	// 2. Required signature on unsigned tag v1.0.0 -> fails
	cfgReqFail := fmt.Sprintf(`schema: gradus/v1
levels:
  - id: team
    source: { git: %s, ref: v1.0.0 }
    signature: required
`, remoteGit)
	if err := os.WriteFile(filepath.Join(projectDir, "gradus.yaml"), []byte(cfgReqFail), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolve.Resolve(projectDir, resolve.Options{}); err == nil || !strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("expected signature verification failure on unsigned ref, got: %v", err)
	}

	// 3. Required signature + min_version 1.1.0 on signed tag v1.2.0 -> passes with signature: verified
	cfgReqPass := fmt.Sprintf(`schema: gradus/v1
levels:
  - id: team
    source: { git: %s, ref: v1.2.0 }
    locked: true
    signature: required
    min_version: 1.1.0
`, remoteGit)
	if err := os.WriteFile(filepath.Join(projectDir, "gradus.yaml"), []byte(cfgReqPass), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = resolve.Resolve(projectDir, resolve.Options{})
	if err != nil {
		t.Fatalf("expected signed v1.2.0 with min_version 1.1.0 to pass, got: %v", err)
	}
	if len(res.Lockfile.Entries) != 1 || res.Lockfile.Entries[0].Signature != "verified" {
		t.Fatalf("expected lockfile signature=verified, got: %+v", res.Lockfile.Entries)
	}

	// 4. min_version 1.3.0 on v1.2.0 -> fails
	cfgMinVerFail := fmt.Sprintf(`schema: gradus/v1
levels:
  - id: team
    source: { git: %s, ref: v1.2.0 }
    signature: required
    min_version: 1.3.0
`, remoteGit)
	if err := os.WriteFile(filepath.Join(projectDir, "gradus.yaml"), []byte(cfgMinVerFail), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolve.Resolve(projectDir, resolve.Options{}); err == nil || !strings.Contains(err.Error(), "below required min_version") {
		t.Fatalf("expected min_version failure, got: %v", err)
	}
}
