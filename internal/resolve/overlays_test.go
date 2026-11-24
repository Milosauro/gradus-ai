package resolve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestConditionalOverlayActivation(t *testing.T) {
	tmp := t.TempDir()

	// Levels: org + project
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "org", "rules"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "level.yaml"), []byte(`schema: gradus/v1
id: org
activation_rules:
  - overlay: gdpr
    when:
      data: personal
      region: [eu, uk]
    required: true
  - overlay: hipaa
    when:
      domain: healthcare
    required: true
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "rules", "ORG-01.md"), []byte("---\nid: ORG-01\ntitle: Org base\nseverity: MUST\nenforcement: prompt\n---\n\nBase org rule.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)

	// Overlays: gdpr (activated by level activation_rules), pci-dss (activated by overlay.yaml when), hipaa (unmatched)
	for _, ov := range []struct {
		id, yaml, ruleID string
	}{
		{"gdpr", "schema: gradus/v1\nid: gdpr\nstrength: org\nactivation: profile\n", "GDPR-01"},
		{"pci-dss", "schema: gradus/v1\nid: pci-dss\nstrength: org\nactivation: profile\nwhen:\n  domain: payments\n  exposure: [public, partner]\n", "PCI-01"},
		{"hipaa", "schema: gradus/v1\nid: hipaa\nstrength: org\nactivation: profile\n", "HIPAA-01"},
	} {
		dir := filepath.Join(tmp, ".gradus", "overlays", ov.id, "rules")
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(tmp, ".gradus", "overlays", ov.id, "overlay.yaml"), []byte(ov.yaml), 0o644)
		_ = os.WriteFile(filepath.Join(dir, ov.ruleID+".md"), []byte("---\nid: "+ov.ruleID+"\ntitle: "+ov.id+"\nseverity: MUST\nenforcement: prompt\n---\n\nOverlay rule.\n"), 0o644)
	}

	// 1. Matching profile activates gdpr (via level:org) and pci-dss (via overlay.yaml), leaves hipaa inactive
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
  - id: pci-dss
    source: { path: .gradus/overlays/pci-dss }
    strength: org
    activation: profile
  - id: hipaa
    source: { path: .gradus/overlays/hipaa }
    strength: org
    activation: profile
profile:
  data: personal
  exposure: PUBLIC
  domain: [payments]
  region: eu
`), 0o644)

	res, err := resolve.Resolve(tmp, resolve.Options{OfflineOnly: true})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(res.View.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got: %+v", res.View.Conflicts)
	}
	ovMap := map[string]resolve.OverlaySummary{}
	for _, o := range res.View.Overlays {
		ovMap[o.ID] = o
	}
	if !ovMap["gdpr"].Active || !strings.Contains(ovMap["gdpr"].Reason, "level:org") {
		t.Fatalf("expected gdpr active via level:org, got %+v", ovMap["gdpr"])
	}
	if !ovMap["pci-dss"].Active || !strings.Contains(ovMap["pci-dss"].Reason, "overlay.yaml") {
		t.Fatalf("expected pci-dss active via overlay.yaml, got %+v", ovMap["pci-dss"])
	}
	if ovMap["hipaa"].Active || ovMap["hipaa"].Reason != "profile mismatch" {
		t.Fatalf("expected hipaa inactive (profile mismatch), got %+v", ovMap["hipaa"])
	}

	ruleIDs := map[string]bool{}
	for _, r := range res.View.Rules {
		ruleIDs[r.ID] = true
	}
	if !ruleIDs["GDPR-01"] || !ruleIDs["PCI-01"] || ruleIDs["HIPAA-01"] {
		t.Fatalf("unexpected resolved rules: %+v", ruleIDs)
	}

	// 2. Mandatory overlay disabled -> blocking overlay conflict
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
`), 0o644)
	resDisabled, err := resolve.Resolve(tmp, resolve.Options{OfflineOnly: true})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(resDisabled.View.Conflicts) != 1 || resDisabled.View.Conflicts[0].Kind != "overlay" {
		t.Fatalf("expected 1 overlay conflict when required overlay is disabled, got: %+v", resDisabled.View.Conflicts)
	}

	// 3. Mandatory overlay missing from gradus.yaml -> blocking overlay conflict
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: org
    source: { path: .gradus/levels/org }
  - id: project
    source: { path: .gradus/levels/project }
profile:
  domain: [healthcare]
`), 0o644)
	resMissing, err := resolve.Resolve(tmp, resolve.Options{OfflineOnly: true})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(resMissing.View.Conflicts) != 1 || resMissing.View.Conflicts[0].Kind != "overlay" || !strings.Contains(resMissing.View.Conflicts[0].Message, "hipaa") {
		t.Fatalf("expected 1 overlay conflict for missing hipaa overlay, got: %+v", resMissing.View.Conflicts)
	}
}
