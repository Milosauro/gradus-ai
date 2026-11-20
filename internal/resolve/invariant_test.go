package resolve_test

import (
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func strPtr(s string) *string { return &s }

func TestRestrictionInvariantAndADR0001(t *testing.T) {
	meta := map[string]resolve.LevelMeta{
		"org":     {ID: "org", Locked: true, EditableRules: []string{"ORG-EDIT-001"}},
		"project": {ID: "project", Locked: false},
		"user":    {ID: "user", Locked: false},
	}

	// 1. Valid specialization on editable rule
	validRules := []resolve.ResolvedRule{
		{ID: "ORG-EDIT-001", Level: "org", Severity: "SHOULD", Enforcement: "prompt", Scope: []string{"services/**"}},
		{ID: "PRJ-EDIT-001", Level: "project", Severity: "MUST", Enforcement: "check", Scope: []string{"services/api/**"}, Overrides: strPtr("ORG-EDIT-001")},
	}
	_, conflicts, err := resolve.MergeAndCheckRules(validRules, meta)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("expected valid specialization to pass, got err=%v conflicts=%+v", err, conflicts)
	}

	// 2. Weakening severity, enforcement, scope, and guard.decision on non-overridable rule
	parent := resolve.ResolvedRule{
		ID:          "ORG-SEC-001",
		Level:       "org",
		Severity:    "MUST",
		Enforcement: "hook",
		Scope:       []string{"services/**"},
		Guard:       &schema.GuardConfig{Action: "command", Pattern: "^git push", Decision: "deny"},
	}
	badChild := resolve.ResolvedRule{
		ID:          "PRJ-SEC-001",
		Level:       "project",
		Severity:    "SHOULD",
		Enforcement: "prompt",
		Scope:       []string{"**/*"},
		Overrides:   strPtr("ORG-SEC-001"),
		Guard:       &schema.GuardConfig{Action: "command", Pattern: "^git push", Decision: "ask"},
	}
	_, badConflicts, err := resolve.MergeAndCheckRules([]resolve.ResolvedRule{parent, badChild}, meta)
	if err != nil {
		t.Fatalf("expected policy conflicts rather than fatal error, got: %v", err)
	}
	// Expect: locked (not in editable_rules) + severity + enforcement + scope + guard.decision = 5 conflicts
	if len(badConflicts) != 5 {
		t.Fatalf("expected 5 conflicts, got %d: %+v", len(badConflicts), badConflicts)
	}

	// 3. Overridable: true allows relaxation when editable
	overridableParent := resolve.ResolvedRule{
		ID:          "ORG-EDIT-001",
		Level:       "org",
		Severity:    "MUST",
		Enforcement: "review",
		Scope:       []string{"services/**"},
		Overridable: &schema.OverridableSpec{Allowed: true},
	}
	relaxedChild := resolve.ResolvedRule{
		ID:          "PRJ-RELAX-001",
		Level:       "project",
		Severity:    "MAY",
		Enforcement: "prompt",
		Scope:       []string{"**/*"},
		Overrides:   strPtr("ORG-EDIT-001"),
	}
	_, okConflicts, err := resolve.MergeAndCheckRules([]resolve.ResolvedRule{overridableParent, relaxedChild}, meta)
	if err != nil || len(okConflicts) != 0 {
		t.Fatalf("expected overridable rule to allow relaxation, got err=%v conflicts=%+v", err, okConflicts)
	}

	// 4. Unknown override target -> fatal error (exit 2)
	_, _, err = resolve.MergeAndCheckRules([]resolve.ResolvedRule{
		{ID: "PRJ-01", Level: "project", Severity: "MUST", Enforcement: "prompt", Scope: []string{"**/*"}, Overrides: strPtr("NO-SUCH-ID")},
	}, meta)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected unknown override target error, got: %v", err)
	}

	// 5. ADR 0001: committed level overriding user-level rule -> fatal error (exit 2)
	_, _, err = resolve.MergeAndCheckRules([]resolve.ResolvedRule{
		{ID: "USR-01", Level: "user", Severity: "SHOULD", Enforcement: "prompt", Scope: []string{"**/*"}},
		{ID: "PRJ-02", Level: "project", Severity: "MUST", Enforcement: "prompt", Scope: []string{"**/*"}, Overrides: strPtr("USR-01")},
	}, meta)
	if err == nil || !strings.Contains(err.Error(), "ADR 0001") {
		t.Fatalf("expected ADR 0001 error, got: %v", err)
	}
}
