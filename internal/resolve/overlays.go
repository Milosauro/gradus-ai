package resolve

import (
	"fmt"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

// LevelActivationRule pairs an activation_rules entry with the level that declared it.
type LevelActivationRule struct {
	LevelID string
	Rule    schema.ActivationRule
}

// MatchProfile evaluates whether profile satisfies every predicate in when.
// Keys are combined with logical AND; list values within a key are combined with logical OR
// using case-insensitive scalar normalization.
func MatchProfile(profile *schema.Profile, when map[string]any) bool {
	if profile == nil || len(when) == 0 {
		return false
	}
	for k, expectedRaw := range when {
		expected := schema.NormalizeProfileValues(expectedRaw)
		if len(expected) == 0 {
			return false
		}
		actual, ok := profile.Lookup(k)
		if !ok || len(actual) == 0 {
			return false
		}
		matched := false
		for _, a := range actual {
			for _, e := range expected {
				if strings.EqualFold(a, e) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// EvaluateOverlayActivation determines whether an overlay is active and returns a human-readable reason.
func EvaluateOverlayActivation(ov schema.OverlayRef, ovCfg *schema.OverlayConfig, profile *schema.Profile, rules []LevelActivationRule) (bool, string) {
	mode := strings.TrimSpace(ov.Activation)
	if mode == "" && ovCfg != nil {
		mode = strings.TrimSpace(ovCfg.Activation)
	}
	switch mode {
	case "":
		return true, "explicit"
	case "always":
		return true, "always"
	case "explicit":
		return true, "explicit"
	case "disabled":
		return false, "disabled"
	case "profile":
		for _, r := range rules {
			if r.Rule.Overlay == ov.ID && MatchProfile(profile, r.Rule.When) {
				return true, fmt.Sprintf("profile match (level:%s)", r.LevelID)
			}
		}
		if ovCfg != nil && MatchProfile(profile, ovCfg.When) {
			return true, "profile match (overlay.yaml)"
		}
		return false, "profile mismatch"
	default:
		return false, fmt.Sprintf("unsupported activation mode %q", mode)
	}
}

// CheckMandatoryOverlays reports blocking overlay conflicts when a matching level activation_rule
// marks an overlay as required: true but it is absent from gradus.yaml or inactive/disabled.
func CheckMandatoryOverlays(profile *schema.Profile, rules []LevelActivationRule, summaries []OverlaySummary) []Conflict {
	declared := make(map[string]bool, len(summaries))
	active := make(map[string]bool, len(summaries))
	for _, s := range summaries {
		declared[s.ID] = true
		active[s.ID] = s.Active
	}

	var conflicts []Conflict
	for _, r := range rules {
		if !r.Rule.Required || !MatchProfile(profile, r.Rule.When) {
			continue
		}
		if !declared[r.Rule.Overlay] {
			conflicts = append(conflicts, Conflict{
				Kind:    "overlay",
				Target:  r.Rule.Overlay,
				Level:   r.LevelID,
				Message: fmt.Sprintf("level %q activation_rules requires overlay %q for the current profile, but it is not declared in gradus.yaml", r.LevelID, r.Rule.Overlay),
			})
			continue
		}
		if !active[r.Rule.Overlay] {
			conflicts = append(conflicts, Conflict{
				Kind:    "overlay",
				Target:  r.Rule.Overlay,
				Level:   r.LevelID,
				Message: fmt.Sprintf("level %q activation_rules requires overlay %q for the current profile, but it is inactive or disabled", r.LevelID, r.Rule.Overlay),
			})
		}
	}
	return conflicts
}
