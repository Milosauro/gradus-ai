package audit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/audit"
	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestAuditCoverageAndExport(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "org", "rules"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)

	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "level.yaml"), []byte("schema: gradus/v1\nid: org\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "rules", "ORG-SEC-01.md"), []byte(`---
id: ORG-SEC-01
title: No hardcoded secrets
severity: MUST
enforcement: check
checker:
  name: regex
  pattern: 'SECRET_'
controls:
  - ISO27001:A.8.28
  - SOC2:CC8.1
---

Never commit secrets.
`), 0o644)

	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	// Override ORG-SEC-01 adding an extra control; must retain parent controls monotonically (R1.AC2)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-SEC-01.md"), []byte(`---
id: PRJ-SEC-01
title: Strict secret scanning
severity: MUST
enforcement: check
overrides: ORG-SEC-01
checker:
  name: regex
  pattern: 'SECRET_|TOKEN_'
controls:
  - OWASP-ASVS:V2.10
---

Never commit secrets or tokens.
`), 0o644)
	// Unmapped rule
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-STYLE-01.md"), []byte(`---
id: PRJ-STYLE-01
title: Use gofmt
severity: SHOULD
enforcement: prompt
---

Format Go files with gofmt.
`), 0o644)

	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: org
    source: { path: .gradus/levels/org }
  - id: project
    source: { path: .gradus/levels/project }
`), 0o644)

	// Write a local evidence record
	rec := evidence.NewRecord("verify", "pass", "abc1234", "main", "sha256:lock", "sha256:view", nil, time.Date(2026, 11, 24, 17, 50, 0, 0, time.UTC))
	evDir := filepath.Join(tmp, ".gradus", "evidence")
	_ = os.MkdirAll(evDir, 0o755)
	recBytes, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(filepath.Join(evDir, rec.ID+".json"), recBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := resolve.Resolve(tmp, resolve.Options{OfflineOnly: true})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify R1.AC2: PRJ-SEC-01 inherited ISO27001:A.8.28 & SOC2:CC8.1 plus OWASP-ASVS:V2.10
	var prjSec *resolve.ResolvedRule
	for i := range res.View.Rules {
		if res.View.Rules[i].ID == "PRJ-SEC-01" {
			prjSec = &res.View.Rules[i]
		}
	}
	if prjSec == nil || len(prjSec.Controls) != 3 {
		t.Fatalf("expected PRJ-SEC-01 to inherit parent controls (total 3), got: %+v", prjSec)
	}

	bundle, err := audit.BuildBundle(tmp, res.View, audit.Options{
		Now: time.Date(2026, 11, 24, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("BuildBundle failed: %v", err)
	}
	if len(bundle.Frameworks) != 3 {
		t.Fatalf("expected 3 frameworks (ISO27001, OWASP-ASVS, SOC2), got %d: %+v", len(bundle.Frameworks), bundle.Frameworks)
	}
	if len(bundle.UnmappedRules) != 1 || bundle.UnmappedRules[0] != "PRJ-STYLE-01" {
		t.Fatalf("expected 1 unmapped rule PRJ-STYLE-01, got %+v", bundle.UnmappedRules)
	}
	if len(bundle.Evidence) != 1 || bundle.Evidence[0].ID != rec.ID {
		t.Fatalf("expected 1 local evidence record, got %+v", bundle.Evidence)
	}

	// Filter by framework (case-insensitive)
	isoBundle, err := audit.BuildBundle(tmp, res.View, audit.Options{Framework: "iso27001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(isoBundle.Frameworks) != 1 || isoBundle.Frameworks[0].Framework != "ISO27001" {
		t.Fatalf("expected only ISO27001 after filter, got %+v", isoBundle.Frameworks)
	}

	md := audit.RenderMarkdown(bundle)
	for _, want := range []string{"# Gradus Compliance & Audit Export", "### ISO27001", "`A.8.28`", "`PRJ-STYLE-01`", rec.ID} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in rendered markdown:\n%s", want, md)
		}
	}
}
