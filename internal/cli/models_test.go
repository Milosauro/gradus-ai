package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/evidence"
)

func TestModelPolicyEvidenceAndVerifyCLI(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Always review\nseverity: MUST\nenforcement: review\n---\n\nReview carefully.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
models:
  tiers:
    tier-high:
      claude-code: opus
    tier-light:
      claude-code: haiku
  operations:
    review: tier-high
targets: [claude-code]
`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve failed: %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile failed: %d %s", code, stderr.String())
	}

	// Verify compiled claude-code skill contains model: opus
	skillBytes, err := os.ReadFile(filepath.Join(tmp, ".claude", "skills", "gradus-review", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skillBytes), "model: opus") {
		t.Fatalf("expected compiled .claude/skills/gradus-review/SKILL.md to contain model: opus, got:\n%s", string(skillBytes))
	}

	// 1. Invalid --model-tier exits 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"evidence", "write", "--dir", tmp, "--kind", "review", "--verdict", "pass", "--skill", "gradus-review", "--model-tier", "tier-bad"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for invalid --model-tier, got %d", code)
	}

	// 2. Write tier-light evidence -> verify --check-model-tier fails with exit 1
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"evidence", "write", "--dir", tmp, "--kind", "review", "--verdict", "pass", "--skill", "gradus-review", "--model-tier", "tier-light", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("evidence write failed: %d %s", code, stderr.String())
	}
	var evOut struct {
		Record evidence.Record `json:"record"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &evOut); err != nil {
		t.Fatal(err)
	}
	if evOut.Record.Producer.Skill == nil || *evOut.Record.Producer.Skill != "gradus-review" || evOut.Record.Producer.ModelTier == nil || *evOut.Record.Producer.ModelTier != "tier-light" {
		t.Fatalf("unexpected evidence producer: %+v", evOut.Record.Producer)
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--check-model-tier", "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected verify --check-model-tier to fail (exit 1) when tier-light < tier-high, got %d\n%s\n%s", code, stdout.String(), stderr.String())
	}

	// 3. Prune old evidence and write tier-high evidence -> verify --check-model-tier passes with exit 0
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"evidence", "prune", "--dir", tmp, "--all"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("evidence prune failed: %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"evidence", "write", "--dir", tmp, "--kind", "review", "--verdict", "pass", "--skill", "gradus-review", "--model-tier", "tier-high"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("evidence write tier-high failed: %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--check-model-tier", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected verify --check-model-tier to pass (exit 0) with tier-high evidence, got %d\n%s\n%s", code, stdout.String(), stderr.String())
	}
}
