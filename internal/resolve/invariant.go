package resolve

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// LevelMeta holds lock and editable_rules metadata for a resolved level.
type LevelMeta struct {
	ID            string
	Locked        bool
	EditableRules []string
}

// MergeAndCheckRules merges ordered rules, enforcing ADR 0001, duplicate detection,
// locked editable_rules, and the P2 restriction invariant.
// Fatal errors (unknown override target, committed-to-user override) are returned as error (exit code 2).
// Policy conflicts (duplicate ID, invariant relaxation, locked non-editable override) are returned as []Conflict (exit code 1).
func MergeAndCheckRules(orderedRules []ResolvedRule, levelMeta map[string]LevelMeta) ([]ResolvedRule, []Conflict, error) {
	var conflicts []Conflict
	byID := make(map[string]ResolvedRule, len(orderedRules))
	var resultOrder []string

	// Track which overlay already overrode a given target ID (D-56).
	overlayOverrides := make(map[string]ResolvedRule)

	for _, r := range orderedRules {
		// Case 1: Specialization via overrides.
		if r.Overrides != nil && *r.Overrides != "" {
			targetID := *r.Overrides
			target, exists := byID[targetID]
			if !exists {
				return nil, nil, fmt.Errorf("rule %q declares overrides %q, which does not exist in upper levels", r.ID, targetID)
			}
			// ADR 0001: committed levels cannot override uncommitted user-level rules.
			if r.Level != "user" && target.Level == "user" {
				return nil, nil, fmt.Errorf("ADR 0001 violation: committed level %q rule %q cannot override user-level rule %q", r.Level, r.ID, targetID)
			}

			// Check locked level editable_rules.
			if meta, ok := levelMeta[target.Level]; ok && meta.Locked {
				if !slices.Contains(meta.EditableRules, targetID) {
					conflicts = append(conflicts, Conflict{
						Kind:    "locked",
						RuleID:  r.ID,
						Target:  targetID,
						Level:   r.Level,
						Message: fmt.Sprintf("rule %q cannot override locked level %q rule %q (not in editable_rules)", r.ID, target.Level, targetID),
					})
				}
			}

			// Check P2 restriction invariant.
			if invConflicts := CheckRestrictionInvariant(r, target); len(invConflicts) > 0 {
				conflicts = append(conflicts, invConflicts...)
			}

			// D-56: Between two active overlays overriding the same ID, keep the more restrictive structural rule.
			if r.IsOverlay {
				if prevOv, dupOv := overlayOverrides[targetID]; dupOv {
					winner, ok := moreRestrictiveRule(prevOv, r)
					if !ok {
						conflicts = append(conflicts, Conflict{
							Kind:    "invariant",
							RuleID:  r.ID,
							Target:  targetID,
							Level:   r.Level,
							Message: fmt.Sprintf("overlays %q and %q conflict while overriding %q with incomparable restrictions", prevOv.ID, r.ID, targetID),
						})
					} else {
						overlayOverrides[targetID] = winner
					}
				} else {
					overlayOverrides[targetID] = r
				}
			}
		}

		// Case 2: Duplicate ID check (if same ID appears without overrides or even with overrides).
		if existing, dup := byID[r.ID]; dup {
			conflicts = append(conflicts, Conflict{
				Kind:    "duplicate",
				RuleID:  r.ID,
				Level:   r.Level,
				Message: fmt.Sprintf("duplicate rule ID %q in level %q (already defined in %q)", r.ID, r.Level, existing.Level),
			})
			continue
		}

		byID[r.ID] = r
		resultOrder = append(resultOrder, r.ID)
	}

	out := make([]ResolvedRule, 0, len(resultOrder))
	for _, id := range resultOrder {
		out = append(out, byID[id])
	}
	return out, conflicts, nil
}

// CheckRestrictionInvariant verifies that child does not relax parent unless parent is overridable (P2).
func CheckRestrictionInvariant(child, parent ResolvedRule) []Conflict {
	if parent.Overridable != nil && parent.Overridable.Allowed {
		return nil
	}

	var conflicts []Conflict

	if severityRank(child.Severity) < severityRank(parent.Severity) {
		conflicts = append(conflicts, Conflict{
			Kind:    "invariant",
			RuleID:  child.ID,
			Target:  parent.ID,
			Level:   child.Level,
			Message: fmt.Sprintf("rule %q lowers severity from %s to %s on non-overridable rule %q", child.ID, parent.Severity, child.Severity, parent.ID),
		})
	}

	if enforcementRank(child.Enforcement) < enforcementRank(parent.Enforcement) {
		conflicts = append(conflicts, Conflict{
			Kind:    "invariant",
			RuleID:  child.ID,
			Target:  parent.ID,
			Level:   child.Level,
			Message: fmt.Sprintf("rule %q weakens enforcement from %s to %s on non-overridable rule %q", child.ID, parent.Enforcement, child.Enforcement, parent.ID),
		})
	}

	if !isScopeSubset(child.Scope, parent.Scope) {
		conflicts = append(conflicts, Conflict{
			Kind:    "invariant",
			RuleID:  child.ID,
			Target:  parent.ID,
			Level:   child.Level,
			Message: fmt.Sprintf("rule %q widens scope %v beyond parent %q scope %v", child.ID, child.Scope, parent.ID, parent.Scope),
		})
	}

	if parent.Guard != nil && child.Guard != nil && parent.Guard.Action == child.Guard.Action {
		if guardDecisionRank(child.Guard.Decision) < guardDecisionRank(parent.Guard.Decision) {
			conflicts = append(conflicts, Conflict{
				Kind:    "guard",
				RuleID:  child.ID,
				Target:  parent.ID,
				Level:   child.Level,
				Message: fmt.Sprintf("rule %q weakens guard decision from %s to %s on %q", child.ID, parent.Guard.Decision, child.Guard.Decision, parent.ID),
			})
		}
	}

	return conflicts
}

func severityRank(s string) int {
	switch s {
	case "MUST":
		return 3
	case "SHOULD":
		return 2
	case "MAY":
		return 1
	default:
		return 0
	}
}

func enforcementRank(e string) int {
	switch e {
	case "permission", "hook":
		return 4
	case "check":
		return 3
	case "review":
		return 2
	case "prompt":
		return 1
	default:
		return 0
	}
}

func guardDecisionRank(d string) int {
	switch d {
	case "deny":
		return 2
	case "ask":
		return 1
	default:
		return 0
	}
}

func isScopeSubset(child, parent []string) bool {
	if len(parent) == 0 {
		return true
	}
	for _, p := range parent {
		if p == "**/*" || p == "**" {
			return true
		}
	}
	if len(child) == 0 {
		return false
	}

	for _, c := range child {
		cNorm := path.Clean(strings.TrimSpace(c))
		covered := false
		for _, p := range parent {
			pNorm := path.Clean(strings.TrimSpace(p))
			if cNorm == pNorm {
				covered = true
				break
			}
			if strings.HasSuffix(pNorm, "/**") {
				prefix := strings.TrimSuffix(pNorm, "/**")
				if strings.HasPrefix(cNorm, prefix+"/") || cNorm == prefix {
					covered = true
					break
				}
			}
			if ok, err := doublestar.Match(pNorm, cNorm); err == nil && ok {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func moreRestrictiveRule(a, b ResolvedRule) (ResolvedRule, bool) {
	aScore := severityRank(a.Severity)*10 + enforcementRank(a.Enforcement)
	bScore := severityRank(b.Severity)*10 + enforcementRank(b.Enforcement)
	if aScore > bScore {
		return a, true
	}
	if bScore > aScore {
		return b, true
	}
	return ResolvedRule{}, false
}
