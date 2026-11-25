package cli_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/compile"
)

func TestManagedSettingsAndChatBundleCLI(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-01.md"), []byte(`---
id: PRJ-01
title: Block rm -rf
severity: MUST
enforcement: permission
guard:
  action: command
  pattern: 'rm -rf *'
  decision: deny
---

Never run rm -rf.
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
anti_tamper:
  mode: deny
levels:
  - id: project
    source: { path: .gradus/levels/project }
toolchain:
  forbidden:
    - type: mcp
      id: "*-pastebin"
targets: [claude-code]
`), 0o644)

	msOut := filepath.Join(tmp, "dist", "managed-settings.json")
	zipOut := filepath.Join(tmp, "dist", "gradus-chat.zip")

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{
		"compile", "--dir", tmp,
		"--managed-settings", msOut,
		"--chat-bundle", zipOut,
		"--json",
	}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile failed: %d %s", code, stderr.String())
	}

	var out struct {
		ManagedSettings string                    `json:"managed_settings"`
		ChatBundle      *compile.ChatBundleResult `json:"chat_bundle"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ManagedSettings != msOut || out.ChatBundle == nil || out.ChatBundle.RuleCount != 1 {
		t.Fatalf("unexpected compile JSON output: %+v", out)
	}

	msBytes, err := os.ReadFile(msOut)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(msBytes), "*-pastebin") || !strings.Contains(string(msBytes), `"disableBypassPermissionsMode": "disable"`) {
		t.Fatalf("unexpected managed-settings.json:\n%s", string(msBytes))
	}

	zr, err := zip.OpenReader(zipOut)
	if err != nil {
		t.Fatal(err)
	}
	_ = zr.Close()

	// Repository verify must still pass with zero false drift from dist/ exports
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected verify to pass cleanly, got %d\n%s\n%s", code, stdout.String(), stderr.String())
	}
}
