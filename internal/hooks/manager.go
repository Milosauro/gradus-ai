package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Supported git hook managers (§11.4).
const (
	ManagerPreCommit = "pre-commit"
	ManagerLefthook  = "lefthook"
	ManagerHusky     = "husky"
	ManagerNative    = "native"
	ManagerNone      = "none"
)

// DetectManager returns the configured manager or auto-detects the manager in use (§11.4).
func DetectManager(rootDir, configured string) string {
	switch strings.TrimSpace(strings.ToLower(configured)) {
	case ManagerPreCommit, ManagerLefthook, ManagerHusky, ManagerNative, ManagerNone:
		return strings.TrimSpace(strings.ToLower(configured))
	}

	if _, err := os.Stat(filepath.Join(rootDir, ".pre-commit-config.yaml")); err == nil {
		return ManagerPreCommit
	}
	if _, err := os.Stat(filepath.Join(rootDir, "lefthook.yml")); err == nil {
		return ManagerLefthook
	}
	if info, err := os.Stat(filepath.Join(rootDir, ".husky")); err == nil && info.IsDir() {
		return ManagerHusky
	}
	return ManagerNative
}

// RenderSnippet returns the configuration snippet for the given git hook manager.
func RenderSnippet(manager string) (string, error) {
	switch manager {
	case ManagerPreCommit:
		return `  - repo: https://github.com/Milosauro/gradus-ai
    rev: v0.1.0
    hooks:
      - id: gradus-verify
      - id: gradus-check
`, nil
	case ManagerLefthook:
		return `pre-commit:
  commands:
    gradus-verify:
      run: gradusctl verify
    gradus-check:
      run: gradusctl check --staged
`, nil
	case ManagerHusky:
		return `gradusctl verify
gradusctl check --staged
`, nil
	case ManagerNative:
		return `#!/bin/sh
set -e
gradusctl verify
gradusctl check --staged
`, nil
	case ManagerNone:
		return "", nil
	default:
		return "", fmt.Errorf("unsupported git hook manager %q", manager)
	}
}

// InstallHooks idempotently installs Gradus git hooks for manager.
// For native hooks, allowNative (--native) must be true before modifying git core.hooksPath (D-72).
func InstallHooks(rootDir, manager string, allowNative bool) (string, error) {
	switch manager {
	case ManagerNone:
		return "", nil
	case ManagerPreCommit:
		p := filepath.Join(rootDir, ".pre-commit-config.yaml")
		snippet, _ := RenderSnippet(ManagerPreCommit)
		existing, _ := os.ReadFile(p)
		if strings.Contains(string(existing), "gradus-verify") {
			return ".pre-commit-config.yaml", nil
		}
		var out string
		if len(existing) == 0 {
			out = "repos:\n" + snippet
		} else {
			out = strings.TrimRight(string(existing), "\n") + "\n" + snippet
		}
		if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
			return "", err
		}
		return ".pre-commit-config.yaml", nil

	case ManagerLefthook:
		p := filepath.Join(rootDir, "lefthook.yml")
		existing, _ := os.ReadFile(p)
		if strings.Contains(string(existing), "gradus-verify") {
			return "lefthook.yml", nil
		}
		snippet, _ := RenderSnippet(ManagerLefthook)
		out := strings.TrimSpace(string(existing))
		if out == "" {
			out = snippet
		} else {
			out = out + "\n\n" + snippet
		}
		if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
			return "", err
		}
		return "lefthook.yml", nil

	case ManagerHusky:
		dir := filepath.Join(rootDir, ".husky")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		p := filepath.Join(dir, "pre-commit")
		existing, _ := os.ReadFile(p)
		if strings.Contains(string(existing), "gradusctl verify") {
			return ".husky/pre-commit", nil
		}
		snippet, _ := RenderSnippet(ManagerHusky)
		out := string(existing)
		if out == "" {
			out = "#!/bin/sh\n" + snippet
		} else {
			out = strings.TrimRight(out, "\n") + "\n" + snippet
		}
		if err := os.WriteFile(p, []byte(out), 0o755); err != nil {
			return "", err
		}
		return ".husky/pre-commit", nil

	case ManagerNative:
		if !allowNative {
			return "", fmt.Errorf("installing native git hooks modifies git core.hooksPath; pass --native to confirm (D-72)")
		}
		dir := filepath.Join(rootDir, ".githooks")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		p := filepath.Join(dir, "pre-commit")
		snippet, _ := RenderSnippet(ManagerNative)
		if err := os.WriteFile(p, []byte(snippet), 0o755); err != nil {
			return "", err
		}
		cmd := exec.Command("git", "config", "core.hooksPath", ".githooks")
		cmd.Dir = rootDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to set git core.hooksPath: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return ".githooks/pre-commit", nil

	default:
		return "", fmt.Errorf("unsupported git hook manager %q", manager)
	}
}
