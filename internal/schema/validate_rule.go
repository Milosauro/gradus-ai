package schema

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var ruleIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(-[A-Z0-9]+)+$`)

// ValidateRule checks required metadata, enums, statement presence, and enforcement requirements.
func ValidateRule(r *Rule) error {
	if !ruleIDPattern.MatchString(r.ID) {
		return fmt.Errorf("invalid rule id %q (must match ^[A-Z][A-Z0-9]*(-[A-Z0-9]+)+$)", r.ID)
	}
	if strings.TrimSpace(r.Title) == "" {
		return fmt.Errorf("rule %s: title is required", r.ID)
	}
	switch r.Severity {
	case "MUST", "SHOULD", "MAY":
	default:
		return fmt.Errorf("rule %s: invalid severity %q (expected MUST, SHOULD, or MAY)", r.ID, r.Severity)
	}
	switch r.Enforcement {
	case "prompt", "review", "check", "hook", "permission":
	default:
		return fmt.Errorf("rule %s: invalid enforcement %q", r.ID, r.Enforcement)
	}
	switch r.Status {
	case "draft", "active", "deprecated", "retired":
	default:
		return fmt.Errorf("rule %s: invalid status %q", r.ID, r.Status)
	}
	if strings.TrimSpace(r.Statement) == "" {
		return fmt.Errorf("rule %s: body must begin with a non-empty statement paragraph", r.ID)
	}
	if err := validateISODate(r.ID, "review_by", r.ReviewBy); err != nil {
		return err
	}
	if err := validateISODate(r.ID, "expires", r.Expires); err != nil {
		return err
	}

	if r.Enforcement == "check" && (r.Checker == nil || strings.TrimSpace(r.Checker.Name) == "") {
		return fmt.Errorf("rule %s: enforcement 'check' requires checker.name", r.ID)
	}
	if r.Enforcement == "hook" {
		if r.Guard == nil || strings.TrimSpace(r.Guard.Action) == "" || strings.TrimSpace(r.Guard.Pattern) == "" {
			return fmt.Errorf("rule %s: enforcement 'hook' requires guard.action and guard.pattern", r.ID)
		}
		if r.Guard.Decision != "deny" && r.Guard.Decision != "ask" {
			return fmt.Errorf("rule %s: guard.decision must be 'deny' or 'ask', got %q", r.ID, r.Guard.Decision)
		}
	}

	// Validate optional dual checker/guard definitions (ADR 0002).
	if r.Checker != nil && strings.TrimSpace(r.Checker.Name) == "" {
		return fmt.Errorf("rule %s: checker.name must not be empty when checker is present", r.ID)
	}
	if r.Guard != nil {
		if strings.TrimSpace(r.Guard.Action) == "" || strings.TrimSpace(r.Guard.Pattern) == "" {
			return fmt.Errorf("rule %s: guard.action and guard.pattern must not be empty when guard is present", r.ID)
		}
		if r.Guard.Decision != "deny" && r.Guard.Decision != "ask" {
			return fmt.Errorf("rule %s: guard.decision must be 'deny' or 'ask', got %q", r.ID, r.Guard.Decision)
		}
	}

	return nil
}

func validateISODate(ruleID, field, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return fmt.Errorf("rule %s: %s %q must be a valid YYYY-MM-DD date", ruleID, field, value)
	}
	return nil
}
