package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/audit"
	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestAuditCLICommands(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-SEC-01.md"), []byte(`---
id: PRJ-SEC-01
title: Pin CI Actions
severity: MUST
enforcement: prompt
controls:
  - ISO27001:A.8.28
  - SOC2:CC8.1
---

Pin all CI actions to 40-char commit SHAs.
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
`), 0o644)

	// 1. audit coverage --json
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"audit", "coverage", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("audit coverage failed: %d %s", code, stderr.String())
	}
	var covOut struct {
		Frameworks []audit.FrameworkCoverage `json:"frameworks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &covOut); err != nil {
		t.Fatal(err)
	}
	if len(covOut.Frameworks) != 2 {
		t.Fatalf("expected 2 frameworks, got %+v", covOut.Frameworks)
	}

	// 2. audit export --format markdown --output
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"audit", "export", "--dir", tmp, "--framework", "ISO27001", "--format", "markdown", "--output", "audit-iso.md"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("audit export markdown failed: %d %s", code, stderr.String())
	}
	mdBytes, err := os.ReadFile(filepath.Join(tmp, "audit-iso.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mdBytes), "### ISO27001") || strings.Contains(string(mdBytes), "### SOC2") {
		t.Fatalf("expected filtered ISO27001 markdown export, got:\n%s", string(mdBytes))
	}

	// 3. audit export --json to stdout
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"audit", "export", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("audit export json failed: %d %s", code, stderr.String())
	}
	var bundle audit.AuditBundle
	if err := json.Unmarshal(stdout.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Schema != "gradus/v1" || len(bundle.Frameworks) != 2 {
		t.Fatalf("unexpected audit bundle: %+v", bundle)
	}

	// 4. invalid --format exits 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"audit", "export", "--dir", tmp, "--format", "xml"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for invalid format xml, got %d", code)
	}
}
