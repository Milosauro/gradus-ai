package schema

import (
	"fmt"
	"regexp"
)

var levelIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidatePyramidConfig validates a parsed gradus.yaml configuration.
func ValidatePyramidConfig(cfg *PyramidConfig) error {
	if cfg.Schema != SchemaVersion {
		return fmt.Errorf("gradus.yaml: unsupported schema %q (expected %q)", cfg.Schema, SchemaVersion)
	}
	if len(cfg.Levels) == 0 {
		return fmt.Errorf("gradus.yaml: levels must contain at least one level")
	}

	seen := make(map[string]bool, len(cfg.Levels))
	for i, lvl := range cfg.Levels {
		if !levelIDPattern.MatchString(lvl.ID) {
			return fmt.Errorf("gradus.yaml: levels[%d].id %q must match ^[a-z][a-z0-9-]*$", i, lvl.ID)
		}
		if seen[lvl.ID] {
			return fmt.Errorf("gradus.yaml: duplicate level id %q", lvl.ID)
		}
		seen[lvl.ID] = true

		if lvl.Source.Path == "" && lvl.Source.Git == "" {
			return fmt.Errorf("gradus.yaml: levels[%d] (%s) source must specify path or git", i, lvl.ID)
		}
		if lvl.Offline != nil {
			switch lvl.Offline.Mode {
			case "cache", "fail-closed":
			default:
				return fmt.Errorf("gradus.yaml: levels[%d] (%s) invalid offline.mode %q", i, lvl.ID, lvl.Offline.Mode)
			}
		}
	}

	seenOverlays := make(map[string]bool, len(cfg.Overlays))
	for i, ov := range cfg.Overlays {
		if !levelIDPattern.MatchString(ov.ID) {
			return fmt.Errorf("gradus.yaml: overlays[%d].id %q must match ^[a-z][a-z0-9-]*$", i, ov.ID)
		}
		if seenOverlays[ov.ID] {
			return fmt.Errorf("gradus.yaml: duplicate overlay id %q", ov.ID)
		}
		seenOverlays[ov.ID] = true
		if !levelIDPattern.MatchString(ov.Strength) {
			return fmt.Errorf("gradus.yaml: overlays[%d].strength %q must match ^[a-z][a-z0-9-]*$", i, ov.Strength)
		}
		if ov.Source.Path == "" && ov.Source.Git == "" {
			return fmt.Errorf("gradus.yaml: overlays[%d] (%s) source must specify path or git", i, ov.ID)
		}
	}

	return nil
}

// ValidateLevelConfig validates a parsed level.yaml configuration.
func ValidateLevelConfig(cfg *LevelConfig) error {
	if cfg.Schema != SchemaVersion {
		return fmt.Errorf("level.yaml: unsupported schema %q (expected %q)", cfg.Schema, SchemaVersion)
	}
	if !levelIDPattern.MatchString(cfg.ID) {
		return fmt.Errorf("level.yaml: id %q must match ^[a-z][a-z0-9-]*$", cfg.ID)
	}
	if cfg.Charter != nil {
		for _, sev := range cfg.Charter.AllowedSeverities {
			switch sev {
			case "MUST", "SHOULD", "MAY":
			default:
				return fmt.Errorf("level.yaml: charter.allowed_severities invalid value %q", sev)
			}
		}
	}
	if cfg.ContextBudget != nil {
		if cfg.ContextBudget.MaxRules != nil && *cfg.ContextBudget.MaxRules < 0 {
			return fmt.Errorf("level.yaml: context_budget.max_rules must be >= 0, got %d", *cfg.ContextBudget.MaxRules)
		}
		if cfg.ContextBudget.MaxChars != nil && *cfg.ContextBudget.MaxChars < 0 {
			return fmt.Errorf("level.yaml: context_budget.max_chars must be >= 0, got %d", *cfg.ContextBudget.MaxChars)
		}
	}
	return nil
}

// ValidateOverlayConfig validates a parsed overlay.yaml configuration.
func ValidateOverlayConfig(cfg *OverlayConfig) error {
	if cfg.Schema != SchemaVersion {
		return fmt.Errorf("overlay.yaml: unsupported schema %q (expected %q)", cfg.Schema, SchemaVersion)
	}
	if !levelIDPattern.MatchString(cfg.ID) {
		return fmt.Errorf("overlay.yaml: id %q must match ^[a-z][a-z0-9-]*$", cfg.ID)
	}
	if !levelIDPattern.MatchString(cfg.Strength) {
		return fmt.Errorf("overlay.yaml: strength %q must match ^[a-z][a-z0-9-]*$", cfg.Strength)
	}
	if cfg.Activation != "" && cfg.Activation != "always" && cfg.Activation != "profile" {
		return fmt.Errorf("overlay.yaml: invalid activation %q (expected always or profile)", cfg.Activation)
	}
	return nil
}
