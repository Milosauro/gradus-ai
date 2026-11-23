package adapter

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

var semverTokenRe = regexp.MustCompile(`v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)`)

// VerifyResult describes verification findings for a single bound adapter.
type VerifyResult struct {
	Provider   string   `json:"provider"`
	TargetFile string   `json:"target_file"`
	Status     string   `json:"status"` // pass | fail
	Drift      string   `json:"drift,omitempty"`
	CLIVersion string   `json:"cli_version,omitempty"`
	Issues     []string `json:"issues,omitempty"`
}

// Verify checks managed-section drift, optional SemVer CLI constraints, and optional provider evidence.
func Verify(rootDir string, m Manifest, view resolve.ResolvedView, requires map[string]string, skipEvidence bool) VerifyResult {
	res := VerifyResult{
		Provider:   m.ID,
		TargetFile: m.TargetFile,
		Status:     "pass",
	}

	absPath, err := ResolveTargetPath(rootDir, m.TargetFile)
	if err != nil {
		res.Status = "fail"
		res.Issues = append(res.Issues, err.Error())
		return res
	}

	raw, readErr := os.ReadFile(absPath)
	if readErr != nil {
		res.Status = "fail"
		if os.IsNotExist(readErr) {
			res.Drift = "missing"
			res.Issues = append(res.Issues, fmt.Sprintf("%s is missing; run `gradusctl adapter sync`", m.TargetFile))
		} else {
			res.Issues = append(res.Issues, fmt.Sprintf("failed to read %s: %v", m.TargetFile, readErr))
		}
	} else {
		expected := RenderConstitution(string(raw), view)
		if !bytes.Equal(raw, expected) {
			res.Status = "fail"
			res.Drift = "modified"
			res.Issues = append(res.Issues, fmt.Sprintf("%s managed section is out of sync; run `gradusctl adapter sync`", m.TargetFile))
		}
	}

	if constraint := strings.TrimSpace(requires["cli"]); constraint != "" {
		ver, verErr := checkCLIVersion(rootDir, m.VersionCommand, constraint)
		res.CLIVersion = ver
		if verErr != nil {
			res.Status = "fail"
			res.Issues = append(res.Issues, verErr.Error())
		}
	}

	if !skipEvidence && len(m.EvidenceCommand) > 0 {
		if out, err := runCommand(rootDir, m.EvidenceCommand); err != nil {
			res.Status = "fail"
			msg := strings.TrimSpace(out)
			if msg != "" {
				res.Issues = append(res.Issues, fmt.Sprintf("evidence command %v failed (%v): %s", m.EvidenceCommand, err, msg))
			} else {
				res.Issues = append(res.Issues, fmt.Sprintf("evidence command %v failed: %v", m.EvidenceCommand, err))
			}
		}
	}

	return res
}

func checkCLIVersion(rootDir string, versionCmd []string, constraint string) (string, error) {
	if len(versionCmd) == 0 {
		return "", fmt.Errorf("cli version constraint %q declared, but adapter defines no version_command", constraint)
	}
	out, err := runCommand(rootDir, versionCmd)
	if err != nil {
		return "", fmt.Errorf("version command %v failed: %w", versionCmd, err)
	}
	match := semverTokenRe.FindStringSubmatch(out)
	if len(match) < 2 {
		return "", fmt.Errorf("could not parse SemVer from version command output %q", strings.TrimSpace(out))
	}
	actualStr := match[1]
	actual, err := schema.ParseSemVer(actualStr)
	if err != nil {
		return actualStr, err
	}

	op, reqStr := splitConstraint(constraint)
	req, err := schema.ParseSemVer(reqStr)
	if err != nil {
		return actualStr, fmt.Errorf("invalid cli version constraint %q: %w", constraint, err)
	}

	cmp := schema.CompareSemVer(actual, req)
	ok := false
	switch op {
	case ">=":
		ok = cmp >= 0
	case ">":
		ok = cmp > 0
	case "<=":
		ok = cmp <= 0
	case "<":
		ok = cmp < 0
	case "=", "==":
		ok = cmp == 0
	default:
		return actualStr, fmt.Errorf("unsupported version operator %q in %q", op, constraint)
	}
	if !ok {
		return actualStr, fmt.Errorf("provider CLI version %s does not satisfy constraint %s", actualStr, constraint)
	}
	return actualStr, nil
}

func splitConstraint(raw string) (string, string) {
	s := strings.TrimSpace(raw)
	for _, op := range []string{">=", "<=", "==", ">", "<", "="} {
		if strings.HasPrefix(s, op) {
			return op, strings.TrimSpace(strings.TrimPrefix(s, op))
		}
	}
	return ">=", s
}

func runCommand(dir string, argv []string) (string, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
