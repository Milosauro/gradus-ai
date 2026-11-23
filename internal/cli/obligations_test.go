package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestObligationsCLIAndHandbook(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "team", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}

	teamLevel := `schema: gradus/v1
id: team
obligations:
  - id: OBL-SDD-01
    rule: "New behavior requires an approved specification before code."
    allowed_providers: [walden, spec-kit]
`
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "team", "level.yaml"), []byte(teamLevel), 0o644); err != nil {
		t.Fatal(err)
	}

	projLevel := `schema: gradus/v1
id: project
bindings:
  - obligation: OBL-SDD-01
    provider: walden
    requires:
      cli: ">=1.4"
    scope: ["services/payments/**"]
`
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte(projLevel), 0o644); err != nil {
		t.Fatal(err)
	}

	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
  - id: project
    source: { path: .gradus/levels/project }
targets: [agents-md]
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. resolve --write --json exposes obligations and bindings
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve failed: %d %s", code, stderr.String())
	}
	var view resolve.ResolvedView
	if err := json.Unmarshal(stdout.Bytes(), &view); err != nil {
		t.Fatalf("expected valid resolve JSON: %v", err)
	}
	if len(view.Obligations) != 1 || !view.Obligations[0].Satisfied || len(view.Bindings) != 1 {
		t.Fatalf("unexpected obligations/bindings in resolve JSON: %+v %+v", view.Obligations, view.Bindings)
	}

	// 2. handbook includes Obligations & Bindings section
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"handbook", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("handbook failed: %d %s", code, stderr.String())
	}
	hb := stdout.String()
	if !strings.Contains(hb, "## Obligations & Bindings") || !strings.Contains(hb, "OBL-SDD-01") || !strings.Contains(hb, "walden") || !strings.Contains(hb, "cli=>=1.4") {
		t.Fatalf("expected Obligations & Bindings details in handbook, got:\n%s", hb)
	}

	// 3. Disallowed provider causes verify to exit 1 with obligation conflict
	badProj := `schema: gradus/v1
id: project
bindings:
  - obligation: OBL-SDD-01
    provider: forbidden-tool
`
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte(badProj), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation || !strings.Contains(stdout.String(), `"kind": "obligation"`) {
		t.Fatalf("expected exit 1 with obligation conflict on verify, got code=%d out=%s", code, stdout.String())
	}
}
