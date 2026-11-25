package assess_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/assess"
	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestProcessMaturityAssessment(t *testing.T) {
	// 1. Minimal repo -> low maturity score (L1/L2)
	minDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(minDir, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(minDir, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(minDir, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644)

	minRep, err := assess.Evaluate(minDir)
	if err != nil {
		t.Fatalf("Evaluate minimal repo failed: %v", err)
	}
	if minRep.TotalScore >= 40 || len(minRep.Dimensions) != 5 || len(minRep.Recommendations) == 0 {
		t.Fatalf("expected low score with recommendations for minimal repo, got %+v", minRep)
	}

	// 2. Mature governed repo -> high maturity score (L4/L5) with zero individual metrics
	matDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(matDir, ".gradus", "levels", "org", "rules"), 0o755)
	_ = os.MkdirAll(filepath.Join(matDir, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(matDir, ".gradus", "levels", "org", "level.yaml"), []byte(`schema: gradus/v1
id: org
charter:
  purpose: Organization security
  allowed_severities: [MUST]
`), 0o644)
	_ = os.WriteFile(filepath.Join(matDir, ".gradus", "levels", "org", "rules", "ORG-01.md"), []byte(`---
id: ORG-01
title: No secrets
severity: MUST
enforcement: check
controls: [ISO27001:A.8.28]
checker:
  name: regex
  pattern: 'SECRET'
---

No secrets.

## Examples
### Compliant
`+"```text\nok\n```"+`
### Non-compliant
`+"```text\nSECRET\n```"+`
`), 0o644)
	_ = os.WriteFile(filepath.Join(matDir, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(matDir, ".gradus", "levels", "project", "rules", "PRJ-01.md"), []byte(`---
id: PRJ-01
title: Ask before push
severity: MUST
enforcement: hook
guard:
  action: command
  pattern: 'git push*'
  decision: ask
---

Ask before push.
`), 0o644)
	_ = os.WriteFile(filepath.Join(matDir, "gradus.yaml"), []byte(`schema: gradus/v1
anti_tamper:
  mode: deny
levels:
  - id: org
    source: { path: .gradus/levels/org }
  - id: project
    source: { path: .gradus/levels/project }
obligations:
  - id: OBL-01
    rule: "Spec required"
    allowed_providers: [walden]
bindings:
  - obligation: OBL-01
    provider: walden
toolchain:
  required:
    - { type: tool, id: git }
models:
  operations:
    review: tier-high
hooks:
  manager: pre-commit
evidence:
  sinks:
    - type: local
targets: [claude-code, antigravity]
`), 0o644)

	resLock, err := resolve.Resolve(matDir, resolve.Options{OfflineOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.WriteLockfile(filepath.Join(matDir, "gradus.lock"), resLock.Lockfile); err != nil {
		t.Fatal(err)
	}
	rec := evidence.NewRecord(matDir, "verify", "pass", "", "sha256:l", "sha256:v", nil, time.Now().UTC())
	if _, err := evidence.WriteRecord(matDir, nil, rec, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	matRep, err := assess.Evaluate(matDir)
	if err != nil {
		t.Fatalf("Evaluate mature repo failed: %v", err)
	}
	if matRep.TotalScore < 85 || matRep.MaturityLevel != "L5 Verified" {
		t.Fatalf("expected L5 Verified (>=85) for mature repo, got %d (%s): %+v", matRep.TotalScore, matRep.MaturityLevel, matRep)
	}

	raw, _ := json.Marshal(matRep)
	if strings.Contains(string(raw), "author") || strings.Contains(string(raw), "developer") {
		t.Fatalf("process maturity assessment must never include individual developer metrics: %s", string(raw))
	}
}
