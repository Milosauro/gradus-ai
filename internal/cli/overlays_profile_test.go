package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestConditionalOverlaysCLI(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "org", "rules"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "overlays", "gdpr", "rules"), 0o755)

	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "level.yaml"), []byte(`schema: gradus/v1
id: org
activation_rules:
  - overlay: gdpr
    when:
      data: personal
      region: eu
    required: true
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "rules", "ORG-01.md"), []byte("---\nid: ORG-01\ntitle: Base\nseverity: MUST\nenforcement: prompt\n---\n\nBase.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "overlays", "gdpr", "overlay.yaml"), []byte("schema: gradus/v1\nid: gdpr\nstrength: org\nactivation: profile\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "overlays", "gdpr", "rules", "GDPR-01.md"), []byte("---\nid: GDPR-01\ntitle: Anonymize PII\nseverity: MUST\nenforcement: prompt\n---\n\nNever commit PII.\n"), 0o644)

	// 1. Matching profile -> resolve --write --json activates gdpr with reason, compile + verify passes
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: org
    source: { path: .gradus/levels/org }
  - id: project
    source: { path: .gradus/levels/project }
overlays:
  - id: gdpr
    source: { path: .gradus/overlays/gdpr }
    strength: org
    activation: profile
profile:
  data: personal
  region: eu
targets: [agents-md]
`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve failed: %d %s", code, stderr.String())
	}
	var view resolve.ResolvedView
	if err := json.Unmarshal(stdout.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Overlays) != 1 || !view.Overlays[0].Active || view.Overlays[0].Reason != "profile match (level:org)" {
		t.Fatalf("expected active gdpr overlay with level:org reason, got %+v", view.Overlays)
	}
	if vals, ok := view.Profile.Lookup("region"); !ok || len(vals) != 1 || vals[0] != "eu" {
		t.Fatalf("expected custom profile field region=eu in JSON resolved view, got %+v", view.Profile)
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile failed: %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("verify failed: %d %s\n%s", code, stderr.String(), stdout.String())
	}

	// 2. Disabling required overlay causes verify to fail with exit 1 (blocking conflict)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: org
    source: { path: .gradus/levels/org }
  - id: project
    source: { path: .gradus/levels/project }
overlays:
  - id: gdpr
    source: { path: .gradus/overlays/gdpr }
    strength: org
    activation: disabled
profile:
  data: personal
  region: eu
targets: [agents-md]
`), 0o644)

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 when mandatory overlay is disabled, got %d\n%s\n%s", code, stdout.String(), stderr.String())
	}
}
