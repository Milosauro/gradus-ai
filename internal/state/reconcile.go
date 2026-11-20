package state

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

// DriftEntry describes a single compiled target file out of sync with the resolved view.
type DriftEntry struct {
	Path    string   `json:"path"`
	Reason  string   `json:"reason"` // missing | modified
	Options []string `json:"options"`
}

// InspectDrift compares expected PlanCompile outputs against on-disk files.
func InspectDrift(rootDir string, targets []string, view resolve.ResolvedView, skillsFS fs.FS) ([]DriftEntry, error) {
	plan, err := compile.PlanCompile(rootDir, view, targets, skillsFS)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(plan))
	for p := range plan {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var out []DriftEntry
	for _, rel := range paths {
		abs := filepath.Join(rootDir, filepath.FromSlash(rel))
		disk, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				out = append(out, DriftEntry{
					Path:    rel,
					Reason:  "missing",
					Options: []string{"restore"},
				})
				continue
			}
			return nil, err
		}
		if !bytes.Equal(disk, plan[rel]) {
			out = append(out, DriftEntry{
				Path:    rel,
				Reason:  "modified",
				Options: []string{"restore", "move"},
			})
		}
	}
	if out == nil {
		return []DriftEntry{}, nil
	}
	return out, nil
}

// RestoreDrift re-applies compile.ApplyCompile to restore all generated targets cleanly.
func RestoreDrift(rootDir string, targets []string, view resolve.ResolvedView, skillsFS fs.FS) (compile.CompileResult, error) {
	return compile.ApplyCompile(rootDir, view, targets, skillsFS)
}

// MoveRuleToLevel validates ruleFile and moves it into targetLevel's rules/ directory.
// Refuses locked levels or remote git levels (D-73 / §11.5).
func MoveRuleToLevel(rootDir string, cfg schema.PyramidConfig, ruleFile, targetLevel string) (string, error) {
	if strings.TrimSpace(ruleFile) == "" || strings.TrimSpace(targetLevel) == "" {
		return "", fmt.Errorf("both rule file and target level are required")
	}

	var targetRef *schema.LevelRef
	for i := range cfg.Levels {
		if cfg.Levels[i].ID == targetLevel {
			targetRef = &cfg.Levels[i]
			break
		}
	}
	if targetRef == nil {
		return "", fmt.Errorf("target level %q not found in gradus.yaml", targetLevel)
	}
	if targetRef.Locked {
		return "", fmt.Errorf("target level %q is locked; cannot move rules into a locked level (§11.5)", targetLevel)
	}
	if strings.TrimSpace(targetRef.Source.Path) == "" {
		return "", fmt.Errorf("target level %q is not a writable local path level", targetLevel)
	}

	srcPath := ruleFile
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(rootDir, filepath.FromSlash(ruleFile))
	}

	// Validate rule frontmatter & statement before moving.
	if _, err := schema.ParseRuleFile(srcPath, targetLevel, targetLevel); err != nil {
		return "", fmt.Errorf("invalid rule file %s: %w", ruleFile, err)
	}

	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", srcPath, err)
	}

	destDir := targetRef.Source.Path
	if !filepath.IsAbs(destDir) {
		destDir = filepath.Join(rootDir, filepath.FromSlash(destDir))
	}
	destRulesDir := filepath.Join(destDir, "rules")
	if err := os.MkdirAll(destRulesDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", destRulesDir, err)
	}

	destPath := filepath.Join(destRulesDir, filepath.Base(srcPath))
	if err := os.WriteFile(destPath, raw, 0o644); err != nil {
		return "", err
	}

	absSrc, _ := filepath.Abs(srcPath)
	absDest, _ := filepath.Abs(destPath)
	if absSrc != absDest {
		_ = os.Remove(srcPath)
	}

	if rel, err := filepath.Rel(rootDir, destPath); err == nil {
		return filepath.ToSlash(rel), nil
	}
	return filepath.ToSlash(destPath), nil
}
