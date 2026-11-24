package check

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

const maxScanLineBytes = 4 * 1024 * 1024 // 4 MiB max line length for large/minified text files

func isBinaryFile(f *os.File) (bool, error) {
	var probe [8192]byte
	n, err := f.Read(probe[:])
	if err != nil && err != io.EOF {
		return false, err
	}
	if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
		return false, seekErr
	}
	return bytes.IndexByte(probe[:n], 0) >= 0, nil
}

// RunBuiltinPatternCheck scans matchedFiles line-by-line against the rule's regex/pattern.
func RunBuiltinPatternCheck(rootDir string, rule resolve.ResolvedRule, matchedFiles []string) ([]evidence.Finding, error) {
	if rule.Checker == nil {
		return nil, fmt.Errorf("rule %s: missing checker configuration", rule.ID)
	}
	rawPattern := rule.Checker.Pattern
	if rawPattern == "" {
		rawPattern = rule.Checker.Config
	}
	if strings.TrimSpace(rawPattern) == "" {
		return nil, fmt.Errorf("rule %s: built-in %s checker requires checker.pattern", rule.ID, rule.Checker.Name)
	}

	re, err := regexp.Compile(rawPattern)
	if err != nil {
		return nil, fmt.Errorf("rule %s: invalid regex pattern %q: %w", rule.ID, rawPattern, err)
	}

	var findings []evidence.Finding
	for _, rel := range matchedFiles {
		abs := filepath.Join(rootDir, filepath.FromSlash(rel))
		f, err := os.Open(abs)
		if err != nil {
			return nil, fmt.Errorf("rule %s: failed to open %s: %w", rule.ID, rel, err)
		}
		bin, binErr := isBinaryFile(f)
		if binErr != nil {
			_ = f.Close()
			return nil, fmt.Errorf("rule %s: failed to inspect %s: %w", rule.ID, rel, binErr)
		}
		if bin {
			_ = f.Close()
			continue
		}

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), maxScanLineBytes)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if re.MatchString(scanner.Text()) {
				findings = append(findings, evidence.Finding{
					RuleID:     rule.ID,
					Severity:   rule.Severity,
					Level:      rule.Level,
					Source:     rule.Provenance.Source,
					Path:       rel,
					Line:       lineNum,
					Message:    fmt.Sprintf("%s (matched pattern %q)", rule.Title, rawPattern),
					Confidence: 1.0,
					Status:     "open",
				})
			}
		}
		scanErr := scanner.Err()
		_ = f.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("rule %s: failed to scan %s: %w", rule.ID, rel, scanErr)
		}
	}

	return findings, nil
}
