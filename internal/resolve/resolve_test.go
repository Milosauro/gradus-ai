package resolve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestFullResolvePipeline(t *testing.T) {
	tmp := t.TempDir()

	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
  - id: project
    source: { path: .gradus/levels/project }
overlays:
  - id: stack-go
    source: { path: .gradus/overlays/stack-go }
    strength: team
    activation: always
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
	ovRules := filepath.Join(tmp, ".gradus", "overlays", "stack-go", "rules")
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	exDir := filepath.Join(tmp, ".gradus", "exceptions")
	findDir := filepath.Join(tmp, ".gradus", "findings")

	for _, d := range []string{teamRules, ovRules, projRules, exDir, findDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeRule := func(dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeRule(teamRules, "TEAM-01.md", "---\nid: TEAM-01\ntitle: Active team rule\nseverity: MUST\n---\nTeam rule.\n")
	writeRule(teamRules, "TEAM-02.md", "---\nid: TEAM-02\ntitle: Retired rule\nseverity: SHOULD\nstatus: retired\n---\nRetired.\n")
	writeRule(teamRules, "TEAM-03.md", "---\nid: TEAM-03\ntitle: Draft rule\nseverity: MAY\nstatus: draft\n---\nDraft.\n")
	writeRule(ovRules, "GO-01.md", "---\nid: GO-01\ntitle: No CGO\nseverity: MUST\n---\nUse CGO_ENABLED=0.\n")
	writeRule(projRules, "PRJ-01.md", "---\nid: PRJ-01\ntitle: Project rule\nseverity: SHOULD\n---\nProject rule.\n")

	// Active exception for TEAM-01 + expired exception for PRJ-01
	if err := os.WriteFile(filepath.Join(exDir, "EX-01.yaml"), []byte("schema: gradus/v1\nid: EX-01\nrule_id: TEAM-01\nscope: ['**/*']\nreason: ok\napproved_by: [lead]\ncreated: 2026-11-01\nexpires: 2026-12-31\nkind: exception\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exDir, "EX-02.yaml"), []byte("schema: gradus/v1\nid: EX-02\nrule_id: PRJ-01\nscope: ['**/*']\nreason: old\napproved_by: [lead]\ncreated: 2026-01-01\nexpires: 2026-02-01\nkind: exception\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Confirmed semantic finding
	if err := os.WriteFile(filepath.Join(findDir, "F-01.yaml"), []byte("schema: gradus/v1\nid: F-01\nrule_a: TEAM-01\nrule_b: PRJ-01\nstatus: confirmed\nmessage: conflicting scopes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	res, err := resolve.Resolve(tmp, resolve.Options{Now: now})
	if err != nil {
		t.Fatalf("expected Resolve to succeed, got: %v", err)
	}

	// Retired & draft omitted -> 3 rules: TEAM-01, GO-01, PRJ-01 (overlay right after team!)
	if len(res.View.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d: %+v", len(res.View.Rules), res.View.Rules)
	}
	if res.View.Rules[0].ID != "TEAM-01" || res.View.Rules[1].ID != "GO-01" || res.View.Rules[2].ID != "PRJ-01" {
		t.Fatalf("unexpected rule ordering (overlay must follow strength 'team'): %+v", res.View.Rules)
	}
	if !res.View.Rules[0].Waived || len(res.View.Rules[0].WaiverIDs) != 1 {
		t.Fatalf("expected TEAM-01 to be waived by EX-01")
	}
	if res.View.Rules[2].Waived {
		t.Fatalf("expected PRJ-01 NOT to be waived by expired EX-02")
	}
	if len(res.View.Conflicts) != 1 || res.View.Conflicts[0].Kind != "semantic" {
		t.Fatalf("expected 1 confirmed semantic conflict, got: %+v", res.View.Conflicts)
	}
	if !strings.HasPrefix(res.View.ViewHash, "sha256:") {
		t.Fatalf("expected valid view_hash, got: %q", res.View.ViewHash)
	}
}
