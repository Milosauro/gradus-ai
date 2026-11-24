package schema

import (
	"fmt"
	"regexp"
	"strings"
)

var levelIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidatePyramidConfig validates a parsed gradus.yaml configuration.
func ValidatePyramidConfig(cfg *PyramidConfig) error {
	if cfg.Schema != SchemaVersion {
		return fmt.Errorf("gradus.yaml: unsupported schema %q (expected %q)", cfg.Schema, SchemaVersion)
	}
	if cfg.AntiTamper != nil {
		switch cfg.AntiTamper.Mode {
		case "", "ask", "deny":
		default:
			return fmt.Errorf("gradus.yaml: invalid anti_tamper.mode %q (expected ask or deny)", cfg.AntiTamper.Mode)
		}
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
		if err := validateSignaturePolicy(fmt.Sprintf("gradus.yaml: levels[%d] (%s)", i, lvl.ID), lvl.Signature); err != nil {
			return err
		}
		if lvl.MinVersion != "" {
			if _, err := ParseSemVer(lvl.MinVersion); err != nil {
				return fmt.Errorf("gradus.yaml: levels[%d] (%s) min_version: %w", i, lvl.ID, err)
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
		if err := validateOverlayActivation("gradus.yaml", ov.Activation); err != nil {
			return err
		}
	}

	if err := validateObligationsAndBindings("gradus.yaml", cfg.Obligations, cfg.Bindings); err != nil {
		return err
	}
	if err := validateToolchainConfig("gradus.yaml", cfg.Toolchain); err != nil {
		return err
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
	if err := validateSignaturePolicy("level.yaml", cfg.Signature); err != nil {
		return err
	}
	if cfg.MinVersion != "" {
		if _, err := ParseSemVer(cfg.MinVersion); err != nil {
			return fmt.Errorf("level.yaml: min_version: %w", err)
		}
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
	if err := validateObligationsAndBindings("level.yaml", cfg.Obligations, cfg.Bindings); err != nil {
		return err
	}
	if err := validateToolchainConfig("level.yaml", cfg.Toolchain); err != nil {
		return err
	}
	for i, ar := range cfg.ActivationRules {
		if !levelIDPattern.MatchString(ar.Overlay) {
			return fmt.Errorf("level.yaml: activation_rules[%d].overlay %q must match ^[a-z][a-z0-9-]*$", i, ar.Overlay)
		}
		if len(ar.When) == 0 {
			return fmt.Errorf("level.yaml: activation_rules[%d].when must declare at least one profile predicate", i)
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
	if err := validateOverlayActivation("overlay.yaml", cfg.Activation); err != nil {
		return err
	}
	return nil
}

func validateOverlayActivation(fileLabel, activation string) error {
	switch activation {
	case "", "always", "explicit", "disabled", "profile":
		return nil
	default:
		return fmt.Errorf("%s: invalid activation %q (expected always, explicit, disabled, or profile)", fileLabel, activation)
	}
}

func validateSignaturePolicy(fileLabel, policy string) error {
	switch policy {
	case "", "optional", "required":
		return nil
	default:
		return fmt.Errorf("%s: invalid signature %q (expected optional or required)", fileLabel, policy)
	}
}

func validateObligationsAndBindings(fileLabel string, obligations []Obligation, bindings []Binding) error {
	seen := make(map[string]bool, len(obligations))
	for i, o := range obligations {
		if strings.TrimSpace(o.ID) == "" {
			return fmt.Errorf("%s: obligations[%d].id is required", fileLabel, i)
		}
		if seen[o.ID] {
			return fmt.Errorf("%s: duplicate obligation id %q", fileLabel, o.ID)
		}
		seen[o.ID] = true
		if strings.TrimSpace(o.Rule) == "" {
			return fmt.Errorf("%s: obligations[%d] (%s) rule is required", fileLabel, i, o.ID)
		}
	}
	for i, b := range bindings {
		if strings.TrimSpace(b.Obligation) == "" {
			return fmt.Errorf("%s: bindings[%d].obligation is required", fileLabel, i)
		}
		if strings.TrimSpace(b.Provider) == "" {
			return fmt.Errorf("%s: bindings[%d] (%s) provider is required", fileLabel, i, b.Obligation)
		}
	}
	return nil
}

func validateToolchainConfig(fileLabel string, tc *ToolchainConfig) error {
	if tc == nil {
		return nil
	}
	checkList := func(section string, items []ToolchainItem) error {
		for i, item := range items {
			switch item.Type {
			case "skill", "tool", "mcp", "plugin":
			default:
				return fmt.Errorf("%s: toolchain.%s[%d].type %q must be skill, tool, mcp, or plugin", fileLabel, section, i, item.Type)
			}
			if strings.TrimSpace(item.ID) == "" {
				return fmt.Errorf("%s: toolchain.%s[%d].id is required", fileLabel, section, i)
			}
		}
		return nil
	}
	if err := checkList("required", tc.Required); err != nil {
		return err
	}
	if err := checkList("recommended", tc.Recommended); err != nil {
		return err
	}
	return checkList("forbidden", tc.Forbidden)
}
