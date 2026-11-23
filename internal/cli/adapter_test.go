package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestAdapterCommand(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}

	rule := `---
id: PRJ-01
title: Require spec before code
severity: MUST
enforcement: review
---

Every new feature requires an approved specification.
`
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-01.md"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}

	gradusYAML := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
obligations:
  - id: OBL-SDD-01
    rule: "New behavior requires an approved specification."
    allowed_providers: [walden, spec-kit]
bindings:
  - obligation: OBL-SDD-01
    provider: spec-kit
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. adapter list --json shows official adapters with spec-kit bound=true
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"adapter", "list", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("adapter list failed: %d %s", code, stderr.String())
	}
	var listResp struct {
		Adapters []struct {
			ID         string `json:"id"`
			Source     string `json:"source"`
			TargetFile string `json:"target_file"`
			Bound      bool   `json:"bound"`
		} `json:"adapters"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &listResp); err != nil {
		t.Fatalf("expected valid adapter list JSON: %v", err)
	}
	if len(listResp.Adapters) < 2 {
		t.Fatalf("expected walden and spec-kit adapters, got: %+v", listResp.Adapters)
	}
	foundBoundSpecKit := false
	for _, a := range listResp.Adapters {
		if a.ID == "spec-kit" && a.Bound && a.Source == "official" {
			foundBoundSpecKit = true
		}
	}
	if !foundBoundSpecKit {
		t.Fatalf("expected spec-kit bound=true in list: %+v", listResp.Adapters)
	}

	// 2. adapter sync writes .specify/memory/constitution.md and is idempotent
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"adapter", "sync", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("adapter sync failed: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("expected changed:true on first sync, got:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"adapter", "sync", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("second adapter sync failed: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"changed": false`) {
		t.Fatalf("expected changed:false on second sync, got:\n%s", stdout.String())
	}

	// 3. adapter verify --skip-evidence passes
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"adapter", "verify", "--dir", tmp, "--skip-evidence", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("adapter verify failed: %d out=%s err=%s", code, stdout.String(), stderr.String())
	}

	// 4. Editing target_file causes adapter verify to fail with exit 1 and clean JSON
	constPath := filepath.Join(tmp, ".specify", "memory", "constitution.md")
	if err := os.WriteFile(constPath, []byte("hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"adapter", "verify", "--dir", tmp, "--skip-evidence", "--json"}, &stdout, &stderr); code != cli.ExitViolation || stderr.Len() != 0 {
		t.Fatalf("expected exit 1 with empty stderr on drift, got code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}

	// 5. Unknown provider -> exit 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"adapter", "verify", "--dir", tmp, "--provider", "unknown-tool"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for unknown provider, got %d", code)
	}
}
