package compile

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// CompileResult summarizes what ApplyCompile wrote, skipped, or pruned.
type CompileResult struct {
	ViewHash       string   `json:"view_hash"`
	Targets        []string `json:"targets"`
	ChangedFiles   []string `json:"changed_files"`
	UnchangedFiles []string `json:"unchanged_files"`
	RemovedFiles   []string `json:"removed_files"`
}

// PlanCompile builds the deterministic in-memory map of relative slash paths to file bytes
// for all configured targets. If targets is empty, defaults to ["agents-md"].
func PlanCompile(rootDir string, view resolve.ResolvedView, targets []string, skillsFS fs.FS) (map[string][]byte, error) {
	if len(targets) == 0 {
		targets = []string{"agents-md"}
	}

	plan := make(map[string][]byte)
	for _, target := range targets {
		switch strings.TrimSpace(strings.ToLower(target)) {
		case "agents-md":
			existing, err := readOptionalFile(filepath.Join(rootDir, "AGENTS.md"))
			if err != nil {
				return nil, err
			}
			plan["AGENTS.md"] = PlanAgentsMD(existing, view)
		case "claude-code":
			sub, err := PlanClaudeCode(rootDir, view, skillsFS)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				plan[k] = v
			}
		case "antigravity":
			sub, err := PlanAntigravity(rootDir, view, skillsFS)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				plan[k] = v
			}
		case "codex":
			sub, err := PlanCodex(rootDir, view, skillsFS)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				plan[k] = v
			}
		case "copilot":
			sub, err := PlanCopilot(rootDir, view, skillsFS)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				plan[k] = v
			}
		case "opencode":
			sub, err := PlanOpenCode(rootDir, view, skillsFS)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				plan[k] = v
			}
		default:
			return nil, fmt.Errorf("unsupported compile target %q", target)
		}
	}

	return plan, nil
}

// ApplyCompile computes PlanCompile, writes changed files idempotently, and prunes stale
// gradus-*.md files under .claude/rules/ and .agents/rules/.
func ApplyCompile(rootDir string, view resolve.ResolvedView, targets []string, skillsFS fs.FS) (CompileResult, error) {
	if len(targets) == 0 {
		targets = []string{"agents-md"}
	}
	res := CompileResult{
		ViewHash:       view.ViewHash,
		Targets:        targets,
		ChangedFiles:   []string{},
		UnchangedFiles: []string{},
		RemovedFiles:   []string{},
	}

	plan, err := PlanCompile(rootDir, view, targets, skillsFS)
	if err != nil {
		return res, err
	}

	paths := make([]string, 0, len(plan))
	for p := range plan {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	claudeKeep := make(map[string]bool)
	agentsKeep := make(map[string]bool)
	copilotKeep := make(map[string]bool)

	for _, rel := range paths {
		if strings.HasPrefix(rel, ".claude/rules/") {
			claudeKeep[filepath.Base(rel)] = true
		}
		if strings.HasPrefix(rel, ".agents/rules/") {
			agentsKeep[filepath.Base(rel)] = true
		}
		if strings.HasPrefix(rel, ".github/instructions/") {
			copilotKeep[filepath.Base(rel)] = true
		}

		absPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		changed, err := WriteIfChanged(absPath, plan[rel])
		if err != nil {
			return res, err
		}
		if changed {
			res.ChangedFiles = append(res.ChangedFiles, rel)
		} else {
			res.UnchangedFiles = append(res.UnchangedFiles, rel)
		}
	}

	for _, target := range targets {
		switch strings.TrimSpace(strings.ToLower(target)) {
		case "claude-code":
			pruned, err := PruneStaleGradusFiles(filepath.Join(rootDir, ".claude", "rules"), "gradus-", claudeKeep)
			if err != nil {
				return res, err
			}
			for _, name := range pruned {
				res.RemovedFiles = append(res.RemovedFiles, ".claude/rules/"+name)
			}
		case "antigravity":
			pruned, err := PruneStaleGradusFiles(filepath.Join(rootDir, ".agents", "rules"), "gradus-", agentsKeep)
			if err != nil {
				return res, err
			}
			for _, name := range pruned {
				res.RemovedFiles = append(res.RemovedFiles, ".agents/rules/"+name)
			}
		case "copilot":
			pruned, err := PruneStaleGradusFiles(filepath.Join(rootDir, ".github", "instructions"), "gradus-", copilotKeep)
			if err != nil {
				return res, err
			}
			for _, name := range pruned {
				res.RemovedFiles = append(res.RemovedFiles, ".github/instructions/"+name)
			}
		}
	}

	return res, nil
}
