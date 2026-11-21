package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/guard"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

// RuleTestCase records the outcome of a single Compliant or Non-compliant example check.
type RuleTestCase struct {
	RuleID   string `json:"rule_id"`
	Kind     string `json:"kind"` // compliant | non-compliant
	Index    int    `json:"index"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message,omitempty"`
}

// RuleTestReport aggregates declarative rule test results across the pyramid.
type RuleTestReport struct {
	Verdict string         `json:"verdict"` // pass | fail
	Passed  int            `json:"passed"`
	Failed  int            `json:"failed"`
	Cases   []RuleTestCase `json:"cases"`
}

// RunRuleTests executes compliant/non-compliant example tests for check and hook rules (§23).
func RunRuleTests(rules []schema.Rule, ruleFilter string) (RuleTestReport, error) {
	rep := RuleTestReport{
		Verdict: "pass",
		Cases:   []RuleTestCase{},
	}

	foundFilter := ruleFilter == ""
	for _, r := range rules {
		if ruleFilter != "" && r.ID != ruleFilter {
			continue
		}
		foundFilter = true
		if r.Status == "draft" || r.Status == "retired" {
			continue
		}
		if len(r.Examples.Compliant) == 0 && len(r.Examples.NonCompliant) == 0 {
			continue
		}

		switch r.Enforcement {
		case "check":
			if r.Checker == nil {
				continue
			}
			cases, err := testCheckRule(r)
			if err != nil {
				return rep, err
			}
			rep.Cases = append(rep.Cases, cases...)
		case "hook":
			if r.Guard == nil {
				continue
			}
			cases, err := testHookRule(r)
			if err != nil {
				return rep, err
			}
			rep.Cases = append(rep.Cases, cases...)
		}
	}

	if !foundFilter {
		return rep, fmt.Errorf("rule %q not found in local pyramid", ruleFilter)
	}

	for _, c := range rep.Cases {
		if c.Passed {
			rep.Passed++
		} else {
			rep.Failed++
		}
	}
	if rep.Failed > 0 {
		rep.Verdict = "fail"
	}
	return rep, nil
}

func testCheckRule(r schema.Rule) ([]RuleTestCase, error) {
	var out []RuleTestCase
	relPath := samplePathForScopes(r.Scope)
	view := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:          r.ID,
				Title:       r.Title,
				Level:       r.Level,
				Severity:    r.Severity,
				Enforcement: r.Enforcement,
				Scope:       r.Scope,
				Checker:     r.Checker,
			},
		},
	}

	runExample := func(kind string, idx int, snippet string, expectFinding bool) (RuleTestCase, error) {
		tmp, err := os.MkdirTemp("", "gradus-ruletest-*")
		if err != nil {
			return RuleTestCase{}, err
		}
		defer os.RemoveAll(tmp)

		target := filepath.Join(tmp, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return RuleTestCase{}, err
		}
		if err := os.WriteFile(target, []byte(stripCodeFence(snippet)), 0o644); err != nil {
			return RuleTestCase{}, err
		}

		res, err := RunChecksOnFiles(tmp, view, []string{relPath})
		if err != nil {
			return RuleTestCase{}, err
		}
		hasFinding := len(res.Findings) > 0
		if hasFinding == expectFinding {
			return RuleTestCase{RuleID: r.ID, Kind: kind, Index: idx, Passed: true}, nil
		}
		msg := fmt.Sprintf("expected findings=%v, got %d finding(s)", expectFinding, len(res.Findings))
		return RuleTestCase{RuleID: r.ID, Kind: kind, Index: idx, Passed: false, Message: msg}, nil
	}

	for i, ex := range r.Examples.Compliant {
		tc, err := runExample("compliant", i+1, ex, false)
		if err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	for i, ex := range r.Examples.NonCompliant {
		tc, err := runExample("non-compliant", i+1, ex, true)
		if err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, nil
}

func testHookRule(r schema.Rule) ([]RuleTestCase, error) {
	var out []RuleTestCase
	actionType := guard.ActionCommand
	switch strings.ToLower(strings.TrimSpace(r.Guard.Action)) {
	case "write", "edit", "write_file", "edit_file":
		actionType = guard.ActionWrite
	}

	resolvedRule := resolve.ResolvedRule{
		ID:          r.ID,
		Title:       r.Title,
		Level:       r.Level,
		Severity:    r.Severity,
		Enforcement: r.Enforcement,
		Scope:       r.Scope,
		Guard:       r.Guard,
	}

	runExample := func(kind string, idx int, snippet string, expectAllow bool) (RuleTestCase, error) {
		target := strings.TrimSpace(stripCodeFence(snippet))
		res, err := guard.Evaluate(guard.NormalizedAction{
			Action: actionType,
			Target: target,
		}, []resolve.ResolvedRule{resolvedRule})
		if err != nil {
			return RuleTestCase{}, err
		}
		isAllow := res.Decision == guard.DecisionAllow
		if isAllow == expectAllow {
			return RuleTestCase{RuleID: r.ID, Kind: kind, Index: idx, Passed: true}, nil
		}
		msg := fmt.Sprintf("expected allow=%v, got decision=%s", expectAllow, res.Decision)
		return RuleTestCase{RuleID: r.ID, Kind: kind, Index: idx, Passed: false, Message: msg}, nil
	}

	for i, ex := range r.Examples.Compliant {
		tc, err := runExample("compliant", i+1, ex, true)
		if err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	for i, ex := range r.Examples.NonCompliant {
		tc, err := runExample("non-compliant", i+1, ex, false)
		if err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, nil
}

func stripCodeFence(s string) string {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n"), "\n")
	if len(lines) >= 2 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		return strings.Join(lines[1:len(lines)-1], "\n") + "\n"
	}
	return s
}

func samplePathForScopes(scopes []string) string {
	if len(scopes) == 0 {
		return "sample.txt"
	}
	s := filepath.ToSlash(strings.TrimSpace(scopes[0]))
	if s == "" || s == "**/*" || s == "**" {
		return "sample.txt"
	}
	s = strings.ReplaceAll(s, "**/", "sub/")
	if strings.Contains(s, "*") {
		ext := filepath.Ext(s)
		dir := filepath.Dir(s)
		dir = strings.ReplaceAll(dir, "*", "sample")
		if ext == "" || strings.Contains(ext, "*") {
			ext = ".txt"
		}
		if dir == "." {
			return "sample" + ext
		}
		return filepath.ToSlash(filepath.Join(dir, "sample"+ext))
	}
	return s
}
