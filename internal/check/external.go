package check

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// RunExternalChecker invokes an external checker binary via os/exec.
// Missing binaries or exit codes >= 2 return an error (mapped to CLI exit code 2, D-73).
// Exit code 1 reports policy violations as structured findings.
func RunExternalChecker(rootDir string, rule resolve.ResolvedRule, matchedFiles []string) ([]evidence.Finding, error) {
	if rule.Checker == nil || strings.TrimSpace(rule.Checker.Name) == "" {
		return nil, fmt.Errorf("rule %s: missing external checker.name", rule.ID)
	}

	binPath, err := exec.LookPath(rule.Checker.Name)
	if err != nil {
		return nil, fmt.Errorf("rule %s: checker binary %q not found on PATH: %w", rule.ID, rule.Checker.Name, err)
	}

	var args []string
	if rule.Checker.Config != "" {
		args = append(args, "--config", rule.Checker.Config)
	}
	args = append(args, matchedFiles...)

	cmd := exec.Command(binPath, args...)
	cmd.Dir = rootDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		return nil, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		msg := strings.TrimSpace(stdout.String())
		if msg == "" {
			msg = strings.TrimSpace(stderr.String())
		}
		if msg == "" {
			msg = fmt.Sprintf("%s reported violations via %s", rule.Title, rule.Checker.Name)
		}
		return []evidence.Finding{{
			RuleID:     rule.ID,
			Severity:   rule.Severity,
			Level:      rule.Level,
			Source:     rule.Provenance.Source,
			Message:    msg,
			Confidence: 1.0,
			Status:     "open",
		}}, nil
	}

	return nil, fmt.Errorf("rule %s: checker %q failed (%v): %s",
		rule.ID, rule.Checker.Name, err, strings.TrimSpace(stderr.String()))
}
