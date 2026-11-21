package check

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// SimulateOptions configures historical impact simulation across git commits.
type SimulateOptions struct {
	Commits int
	Since   string
	RuleID  string
}

// SimulatedCommit records findings triggered on a single historical commit.
type SimulatedCommit struct {
	SHA      string             `json:"sha"`
	Findings []evidence.Finding `json:"findings"`
}

// SimulateReport summarizes historical rule impact across scanned commits.
type SimulateReport struct {
	CommitsScanned  int               `json:"commits_scanned"`
	AffectedCommits int               `json:"affected_commits"`
	FindingsByRule  map[string]int    `json:"findings_by_rule"`
	Commits         []SimulatedCommit `json:"commits"`
}

// RunSimulation evaluates active check rules against historical commits via os/exec git (PRJ-GO-001).
func RunSimulation(rootDir string, view resolve.ResolvedView, opts SimulateOptions) (SimulateReport, error) {
	rep := SimulateReport{
		FindingsByRule: map[string]int{},
		Commits:        []SimulatedCommit{},
	}

	if strings.HasPrefix(strings.TrimSpace(opts.Since), "-") {
		return rep, fmt.Errorf("invalid --since ref %q: must not start with '-'", opts.Since)
	}

	filteredView := view
	if opts.RuleID != "" {
		var filtered []resolve.ResolvedRule
		for _, r := range view.Rules {
			if r.ID == opts.RuleID {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			return rep, fmt.Errorf("rule %q not found in resolved view", opts.RuleID)
		}
		filteredView.Rules = filtered
	}

	commits, err := listCommitsForSimulation(rootDir, opts)
	if err != nil {
		return rep, err
	}
	rep.CommitsScanned = len(commits)

	for _, sha := range commits {
		files, err := listCommitFiles(rootDir, sha)
		if err != nil {
			return rep, err
		}
		if len(files) == 0 {
			continue
		}

		tmp, err := os.MkdirTemp("", "gradus-sim-*")
		if err != nil {
			return rep, err
		}

		var materialized []string
		for _, rel := range files {
			blob, err := gitShowBlob(rootDir, sha, rel)
			if err != nil {
				continue
			}
			dest := filepath.Join(tmp, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				_ = os.RemoveAll(tmp)
				return rep, err
			}
			if err := os.WriteFile(dest, blob, 0o644); err != nil {
				_ = os.RemoveAll(tmp)
				return rep, err
			}
			materialized = append(materialized, rel)
		}

		res, err := RunChecksOnFiles(tmp, filteredView, materialized)
		_ = os.RemoveAll(tmp)
		if err != nil {
			return rep, err
		}

		if len(res.Findings) > 0 {
			rep.AffectedCommits++
			for _, f := range res.Findings {
				rep.FindingsByRule[f.RuleID]++
			}
			rep.Commits = append(rep.Commits, SimulatedCommit{
				SHA:      sha,
				Findings: res.Findings,
			})
		}
	}

	return rep, nil
}

func listCommitsForSimulation(rootDir string, opts SimulateOptions) ([]string, error) {
	var args []string
	if strings.TrimSpace(opts.Since) != "" {
		args = []string{"rev-list", opts.Since + "..HEAD"}
	} else {
		n := opts.Commits
		if n <= 0 {
			n = 10
		}
		args = []string{"rev-list", "-n", strconv.Itoa(n), "HEAD"}
	}
	out, err := runGitCommand(rootDir, args...)
	if err != nil {
		return nil, err
	}
	var shas []string
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if s := strings.TrimSpace(line); s != "" {
			shas = append(shas, s)
		}
	}
	return shas, nil
}

func listCommitFiles(rootDir, sha string) ([]string, error) {
	out, err := runGitCommand(rootDir, "diff-tree", "--no-commit-id", "--name-only", "-r", "--diff-filter=ACMR", sha)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if f := strings.TrimSpace(line); f != "" {
			files = append(files, filepath.ToSlash(f))
		}
	}
	return files, nil
}

func gitShowBlob(rootDir, sha, relPath string) ([]byte, error) {
	return runGitCommand(rootDir, "show", sha+":"+filepath.ToSlash(relPath))
}

func runGitCommand(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %v failed: %w (%s)", args, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
