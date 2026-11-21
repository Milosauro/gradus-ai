package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

// Guard decision values ordered by precedence: deny > ask > allow.
const (
	DecisionAllow = "allow"
	DecisionAsk   = "ask"
	DecisionDeny  = "deny"
)

// MatchedRule identifies a hook rule that matched the evaluated invocation.
type MatchedRule struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Level    string `json:"level"`
	Decision string `json:"decision"`
}

// HookSpecificOutput provides PreToolUse metadata compatible with Claude Code and Antigravity.
type HookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason"`
}

// EvaluationResult holds the final guard verdict, matched rules, and agent-compatible output.
type EvaluationResult struct {
	Decision           string              `json:"decision"`
	Reason             string              `json:"reason"`
	MatchedRules       []MatchedRule       `json:"matched_rules"`
	AntiTamper         bool                `json:"anti_tamper"`
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput"`
}

// LoadLocalGuardRules resolves the local pyramid offline (zero network calls) and appends
// personal user-level rules from userLevelDir (defaulting to ~/.gradus/levels/user) when present (ADR 0003).
func LoadLocalGuardRules(rootDir, userLevelDir string) ([]resolve.ResolvedRule, error) {
	return LoadLocalGuardRulesWithProfile(rootDir, userLevelDir, "")
}

// LoadLocalGuardRulesWithProfile resolves the local pyramid offline and optionally replaces
// the personal user level with a team-defined agent profile directory (§21).
func LoadLocalGuardRulesWithProfile(rootDir, userLevelDir, agentProfile string) ([]resolve.ResolvedRule, error) {
	res, err := resolve.Resolve(rootDir, resolve.Options{
		OfflineOnly:  true,
		AgentProfile: agentProfile,
	})
	if err != nil {
		return nil, err
	}

	rules := append([]resolve.ResolvedRule(nil), res.View.Rules...)
	activeProfile := strings.TrimSpace(agentProfile) != "" || strings.TrimSpace(os.Getenv("GRADUS_AGENT_PROFILE")) != ""

	if !activeProfile && !res.View.Scope.IncludesUserLevel {
		dir := strings.TrimSpace(userLevelDir)
		if dir == "" && os.Getenv("CI") == "" {
			if home, homeErr := os.UserHomeDir(); homeErr == nil {
				dir = filepath.Join(home, ".gradus", "levels", "user")
			}
		}
		if dir != "" {
			userRules, err := loadUserLevelRules(dir, res.View.Exceptions)
			if err != nil {
				return nil, err
			}
			rules = append(rules, userRules...)
		}
	}

	return rules, nil
}

func loadUserLevelRules(userDir string, exceptions []resolve.ExceptionView) ([]resolve.ResolvedRule, error) {
	rulesDir := filepath.Join(userDir, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read user rules dir %s: %w", rulesDir, err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var out []resolve.ResolvedRule
	for _, name := range names {
		fullPath := filepath.Join(rulesDir, name)
		if linfo, lerr := os.Lstat(fullPath); lerr == nil && linfo.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink user rule file %s is not allowed (Threat Model T2)", fullPath)
		}
		rawBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		parsed, err := schema.ParseRuleBytes(rawBytes, fullPath, "user", "user")
		if err != nil {
			return nil, fmt.Errorf("user level rule %s: %w", name, err)
		}
		r := parsed.Value
		if r.Status != "active" {
			continue
		}

		waived := false
		var waiverIDs []string
		for _, ex := range exceptions {
			if ex.RuleID == r.ID {
				waived = true
				waiverIDs = append(waiverIDs, ex.ID)
			}
		}

		out = append(out, resolve.ResolvedRule{
			ID:               r.ID,
			Title:            r.Title,
			Level:            "user",
			Severity:         r.Severity,
			Enforcement:      r.Enforcement,
			Scope:            r.Scope,
			Text:             r.Statement,
			Why:              r.Why,
			Rationalizations: r.Rationalizations,
			Overridable:      r.Overridable,
			Checker:          r.Checker,
			Guard:            r.Guard,
			Waived:           waived,
			WaiverIDs:        waiverIDs,
			Provenance: resolve.RuleProvenance{
				Source: "user",
				Path:   filepath.ToSlash(filepath.Join("rules", name)),
				Hash:   lock.HashBytes(rawBytes),
			},
		})
	}
	return out, nil
}

// EvaluateOptions configures guard evaluation behavior.
type EvaluateOptions struct {
	AntiTamperMode string // "ask" (default) or "deny"
}

// LoadAntiTamperMode reads anti_tamper.mode from rootDir/gradus.yaml, defaulting to "ask".
func LoadAntiTamperMode(rootDir string) string {
	raw, err := os.ReadFile(filepath.Join(rootDir, "gradus.yaml"))
	if err != nil {
		return DecisionAsk
	}
	parsed, err := schema.ParsePyramidConfig(raw)
	if err != nil || parsed.Value.AntiTamper == nil || parsed.Value.AntiTamper.Mode == "" {
		return DecisionAsk
	}
	if parsed.Value.AntiTamper.Mode == DecisionDeny {
		return DecisionDeny
	}
	return DecisionAsk
}

// Evaluate applies built-in anti-tamper protection (T2) and active hook rules using
// strict precedence: deny > ask > allow.
func Evaluate(action NormalizedAction, rules []resolve.ResolvedRule) (EvaluationResult, error) {
	return EvaluateWithOptions(action, rules, EvaluateOptions{AntiTamperMode: DecisionAsk})
}

// EvaluateWithOptions applies built-in anti-tamper protection (T2) and active hook rules
// using strict precedence: deny > ask > allow.
func EvaluateWithOptions(action NormalizedAction, rules []resolve.ResolvedRule, opts EvaluateOptions) (EvaluationResult, error) {
	res := EvaluationResult{
		Decision:     DecisionAllow,
		Reason:       "allowed by guard",
		MatchedRules: []MatchedRule{},
	}

	tamperDecision := DecisionAsk
	if opts.AntiTamperMode == DecisionDeny {
		tamperDecision = DecisionDeny
	}

	var reasons []string

	// Built-in anti-tamper floor (Threat Model T2) for both file writes and mutating shell commands.
	if action.Action == ActionWrite && IsProtectedGovernancePath(action.Target) {
		res.AntiTamper = true
		res.Decision = tamperDecision
		reasons = append(reasons, fmt.Sprintf("anti-tamper: modifying protected governance file %q requires explicit human confirmation", action.Target))
	} else if action.Action == ActionCommand && IsProtectedShellCommand(action.Target) {
		res.AntiTamper = true
		res.Decision = tamperDecision
		reasons = append(reasons, "anti-tamper: shell command targets protected governance path and requires explicit human confirmation")
	}

	for _, r := range rules {
		if r.Waived || r.Enforcement != "hook" || r.Guard == nil {
			continue
		}
		if !guardMatchesAction(r.Guard.Action, action.Action) {
			continue
		}
		if action.Action == ActionWrite && !matchesRuleScope(r.Scope, action.Target) {
			continue
		}

		matched, err := matchGuardPattern(action.Action, r.Guard.Pattern, action.Target)
		if err != nil {
			return EvaluationResult{}, fmt.Errorf("rule %s: invalid guard pattern %q: %w", r.ID, r.Guard.Pattern, err)
		}
		if !matched {
			continue
		}

		dec := normalizeDecision(r.Guard.Decision)
		res.MatchedRules = append(res.MatchedRules, MatchedRule{
			ID:       r.ID,
			Title:    r.Title,
			Level:    r.Level,
			Decision: dec,
		})
		reasons = append(reasons, fmt.Sprintf("%s (%s): %s", r.ID, dec, r.Title))

		if decisionRank(dec) > decisionRank(res.Decision) {
			res.Decision = dec
		}
	}

	if len(reasons) > 0 {
		res.Reason = strings.Join(reasons, "; ")
	}
	res.HookSpecificOutput = &HookSpecificOutput{
		HookEventName:            "PreToolUse",
		PermissionDecision:       res.Decision,
		PermissionDecisionReason: res.Reason,
	}

	return res, nil
}

func guardMatchesAction(guardAction, normalizedAction string) bool {
	ga := strings.ToLower(strings.TrimSpace(guardAction))
	switch normalizedAction {
	case ActionCommand:
		return ga == "command" || ga == "bash"
	case ActionWrite:
		return ga == "write" || ga == "edit"
	default:
		return false
	}
}

func matchesRuleScope(scope []string, relPath string) bool {
	if len(scope) == 0 {
		return true
	}
	for _, pat := range scope {
		if pat == "**/*" || pat == "**" {
			return true
		}
		if ok, err := doublestar.Match(pat, relPath); err == nil && ok {
			return true
		}
	}
	return false
}

func matchGuardPattern(actionClass, pattern, target string) (bool, error) {
	if actionClass == ActionWrite {
		if ok, err := doublestar.Match(pattern, target); err == nil && ok {
			return true, nil
		}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(target), nil
}

func normalizeDecision(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case DecisionDeny:
		return DecisionDeny
	case DecisionAsk:
		return DecisionAsk
	default:
		return DecisionAllow
	}
}

func decisionRank(d string) int {
	switch d {
	case DecisionDeny:
		return 2
	case DecisionAsk:
		return 1
	default:
		return 0
	}
}
