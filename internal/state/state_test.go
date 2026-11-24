package state_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/state"
)

func TestFindingsAndCandidates(t *testing.T) {
	tmp := t.TempDir()
	now := time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC)

	view := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:         "TEAM-LOG-01",
				Title:      "Use structured JSON logging",
				Level:      "team",
				Scope:      []string{"internal/**"},
				Text:       "Emit structured JSON logs in backend services.",
				Provenance: resolve.RuleProvenance{Source: "team@v1", Hash: "sha256:aaa"},
			},
			{
				ID:         "PRJ-LOG-01",
				Title:      "Use plain text logging in CLI",
				Level:      "project",
				Scope:      []string{"**/*"},
				Text:       "Print plain text logs instead of JSON logs.",
				Provenance: resolve.RuleProvenance{Source: "project", Hash: "sha256:bbb"},
			},
			{
				ID:         "PRJ-OTHER-01",
				Title:      "Format markdown tables",
				Level:      "project",
				Scope:      []string{"docs/**"},
				Text:       "Align pipes in markdown tables.",
				Provenance: resolve.RuleProvenance{Source: "project", Hash: "sha256:ccc"},
			},
		},
	}

	// 1. Candidate discovery finds TEAM-LOG-01 <-> PRJ-LOG-01 (canonical alphabetical order)
	existing, err := state.ListFindings(tmp)
	if err != nil {
		t.Fatal(err)
	}
	cands := state.FindCandidates(view, existing)
	if len(cands) != 1 || cands[0].RuleA != "PRJ-LOG-01" || cands[0].RuleB != "TEAM-LOG-01" {
		t.Fatalf("expected 1 candidate pair (PRJ-LOG-01, TEAM-LOG-01), got: %+v", cands)
	}

	// 2. Dismissing with current hashes suppresses the pair from subsequent --candidates
	_, _, err = state.SaveFinding(tmp, state.FindingRecord{
		RuleA:   "TEAM-LOG-01",
		HashA:   "sha256:aaa",
		RuleB:   "PRJ-LOG-01",
		HashB:   "sha256:bbb",
		Status:  "dismissed",
		Message: "CLI and backend scopes differ in practice",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	existing, err = state.ListFindings(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if cands2 := state.FindCandidates(view, existing); len(cands2) != 0 {
		t.Fatalf("expected dismissed pair with unchanged hashes to be suppressed, got: %+v", cands2)
	}

	// 3. Changing one rule's content hash re-surfaces the pair
	view.Rules[1].Provenance.Hash = "sha256:bbb-updated"
	if cands3 := state.FindCandidates(view, existing); len(cands3) != 1 {
		t.Fatalf("expected modified rule hash to re-surface candidate pair, got: %+v", cands3)
	}

	// 4. Confirming updates the canonical finding record
	rec, _, err := state.SaveFinding(tmp, state.FindingRecord{
		RuleA:   "PRJ-LOG-01",
		HashA:   "sha256:bbb-updated",
		RuleB:   "TEAM-LOG-01",
		HashB:   "sha256:aaa",
		Status:  "confirmed",
		Message: "Confirmed conflict",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Status != "confirmed" || rec.ID != "F-PRJ-LOG-01-TEAM-LOG-01" {
		t.Fatalf("unexpected confirmed finding: %+v", rec)
	}
}

func TestLessonsAndRedaction(t *testing.T) {
	tmp := t.TempDir()
	now := time.Date(2026, 11, 20, 14, 30, 0, 0, time.UTC)

	// 1. ProjectKey is deterministic
	if state.ProjectKey(tmp) != state.ProjectKey(filepath.Clean(tmp)) {
		t.Fatalf("expected deterministic ProjectKey")
	}

	// 2. AddLesson redacts secrets & PII (including PEM keys, Codice Fiscale, mobile numbers) without false positives
	rec, dest, err := state.AddLesson(tmp, state.LessonRecord{
		Kind:      "correction",
		Statement: "Never hardcode ghp_1234567890abcdef123456, sk-proj-abc1234567890xyz, AKIAIOSFODNN7EXAMPLE, password=hunter2, IT60X0542811101000000123456, RSSMRA80A01H501U, +39 333 1234567, or alice@example.com (keep numbers 100 200 300 400 intact)",
		Evidence: []string{
			"Authorization: Bearer secret_token_value_123456",
			"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAtest\n-----END RSA PRIVATE KEY-----",
		},
	}, now)
	if err != nil {
		t.Fatalf("AddLesson failed: %v", err)
	}
	raw, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	diskStr := string(raw)
	for _, forbidden := range []string{"ghp_", "sk-proj-", "AKIAIOSFODNN7EXAMPLE", "hunter2", "IT60X0542811101000000123456", "RSSMRA80A01H501U", "333 1234567", "alice@example.com", "secret_token_value_123456", "BEGIN RSA PRIVATE KEY"} {
		if strings.Contains(diskStr, forbidden) {
			t.Fatalf("expected %q to be redacted on disk, got:\n%s", forbidden, diskStr)
		}
	}
	if !strings.Contains(diskStr, "100 200 300 400") {
		t.Fatalf("expected plain space-separated numbers not to trigger phone false positive, got:\n%s", diskStr)
	}
	if strings.Count(diskStr, "[REDACTED]") < 10 {
		t.Fatalf("expected redacted placeholders on disk, got:\n%s", diskStr)
	}

	// 3. List & UpdateLesson
	listed, err := state.ListLessons(tmp)
	if err != nil || len(listed) != 1 || listed[0].ID != rec.ID {
		t.Fatalf("unexpected ListLessons output: %+v (err=%v)", listed, err)
	}
	updated, err := state.UpdateLesson(tmp, rec.ID, "promoted", true)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "promoted" || updated.Occurrences != 2 {
		t.Fatalf("unexpected updated lesson: %+v", updated)
	}
}

func TestReconcileState(t *testing.T) {
	tmp := t.TempDir()
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
	for _, d := range []string{projRules, teamRules} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cfg := schema.PyramidConfig{
		Schema: schema.SchemaVersion,
		Levels: []schema.LevelRef{
			{ID: "team", Source: schema.SourceRef{Path: ".gradus/levels/team"}, Locked: true},
			{ID: "project", Source: schema.SourceRef{Path: ".gradus/levels/project"}, Locked: false},
		},
		Targets: []string{"agents-md"},
	}
	view := resolve.ResolvedView{
		ViewHash: "sha256:rec1",
		Levels:   []resolve.LevelSummary{{ID: "project"}},
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-01",
				Title:       "Project rule",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "Stay consistent.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
		},
	}
	memFS := fstest.MapFS{}

	// 1. Compile, then introduce drift in AGENTS.md
	if _, err := compile.ApplyCompile(tmp, view, cfg.Targets, memFS); err != nil {
		t.Fatal(err)
	}
	agentsPath := filepath.Join(tmp, "AGENTS.md")
	_ = os.WriteFile(agentsPath, []byte("drifted\n"), 0o644)

	drift, err := state.InspectDrift(tmp, cfg.Targets, view, memFS)
	if err != nil || len(drift) != 1 || drift[0].Path != "AGENTS.md" {
		t.Fatalf("expected drift on AGENTS.md, got %+v (err=%v)", drift, err)
	}

	// 2. RestoreDrift clears drift
	if _, err := state.RestoreDrift(tmp, cfg.Targets, view, memFS); err != nil {
		t.Fatal(err)
	}
	driftAfter, err := state.InspectDrift(tmp, cfg.Targets, view, memFS)
	if err != nil || len(driftAfter) != 0 {
		t.Fatalf("expected 0 drift after RestoreDrift, got %+v (err=%v)", driftAfter, err)
	}

	// 3. MoveRuleToLevel succeeds on unlocked project level and refuses locked team level
	draftRule := filepath.Join(tmp, "NEW-01.md")
	_ = os.WriteFile(draftRule, []byte("---\nid: NEW-01\ntitle: Draft rule\nseverity: SHOULD\n---\nFollow convention.\n"), 0o644)

	if _, err := state.MoveRuleToLevel(tmp, cfg, draftRule, "team"); err == nil {
		t.Fatalf("expected MoveRuleToLevel to refuse locked team level")
	}
	destRel, err := state.MoveRuleToLevel(tmp, cfg, draftRule, "project")
	if err != nil {
		t.Fatalf("expected MoveRuleToLevel to succeed on unlocked project level, got: %v", err)
	}
	if destRel != ".gradus/levels/project/rules/NEW-01.md" {
		t.Fatalf("unexpected destination path: %q", destRel)
	}
}

func TestRedactionGapsV032(t *testing.T) {
	// Unspaced 10-digit Italian mobile number
	phoneOut := state.RedactText("Call me at 3331234567 before release.")
	if strings.Contains(phoneOut, "3331234567") || !strings.Contains(phoneOut, "[REDACTED]") {
		t.Fatalf("expected unspaced mobile number to be redacted, got %q", phoneOut)
	}

	// Benign non-mobile number (e.g. timestamp) should not be redacted
	tsOut := state.RedactText("Timestamp 1732450000 recorded.")
	if !strings.Contains(tsOut, "1732450000") {
		t.Fatalf("expected unix timestamp not to be redacted, got %q", tsOut)
	}

	// Truncated PEM private key without END footer
	truncPEM := "Leaked key:\n-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA7V5x..."
	pemOut := state.RedactText(truncPEM)
	if strings.Contains(pemOut, "MIIEpAIBAAKCAQEA") || !strings.Contains(pemOut, "[REDACTED]") {
		t.Fatalf("expected truncated PEM private key to be redacted, got %q", pemOut)
	}
}
