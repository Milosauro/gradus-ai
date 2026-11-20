package check

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// Options configures file discovery for deterministic checks.
type Options struct {
	Staged bool
	Base   string
}

// CheckResult holds the discovered files and findings produced by RunChecks.
type CheckResult struct {
	Files    []string           `json:"files"`
	Findings []evidence.Finding `json:"findings"`
}

// DiscoverFiles returns relative slash paths of files to check via git subprocesses.
func DiscoverFiles(rootDir string, opts Options) ([]string, error) {
	var args []string
	switch {
	case opts.Staged:
		args = []string{"diff", "--cached", "--name-only", "--diff-filter=ACMR"}
	case opts.Base != "":
		args = []string{"diff", "--name-only", "--diff-filter=ACMR", opts.Base + "...HEAD", "--"}
	default:
		args = []string{"ls-files"}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = rootDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %v failed: %w (%s)", args, err, strings.TrimSpace(stderr.String()))
	}

	var files []string
	for _, line := range strings.Split(strings.ReplaceAll(stdout.String(), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			files = append(files, filepath.ToSlash(trimmed))
		}
	}
	return files, nil
}

// RunChecks executes all active, non-waived check rules matching the discovered files.
func RunChecks(rootDir string, view resolve.ResolvedView, opts Options) (CheckResult, error) {
	files, err := DiscoverFiles(rootDir, opts)
	if err != nil {
		return CheckResult{}, err
	}

	res := CheckResult{
		Files:    files,
		Findings: []evidence.Finding{},
	}
	if len(files) == 0 {
		return res, nil
	}

	for _, rule := range view.Rules {
		if rule.Waived || rule.Enforcement != "check" || rule.Checker == nil {
			continue
		}

		matched := filterScopedFiles(files, rule.Scope)
		if len(matched) == 0 {
			continue
		}

		var findings []evidence.Finding
		switch strings.ToLower(strings.TrimSpace(rule.Checker.Name)) {
		case "pattern", "regex":
			findings, err = RunBuiltinPatternCheck(rootDir, rule, matched)
		default:
			findings, err = RunExternalChecker(rootDir, rule, matched)
		}
		if err != nil {
			return res, err
		}
		res.Findings = append(res.Findings, findings...)
	}

	return res, nil
}

func filterScopedFiles(files []string, scopes []string) []string {
	if len(scopes) == 0 {
		return files
	}
	var out []string
	for _, f := range files {
		for _, s := range scopes {
			if s == "**/*" || s == "**" {
				out = append(out, f)
				break
			}
			if ok, err := doublestar.Match(s, f); err == nil && ok {
				out = append(out, f)
				break
			}
		}
	}
	return out
}
