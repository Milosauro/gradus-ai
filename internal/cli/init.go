package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type initResult struct {
	CreatedFiles []string `json:"created_files"`
	UpdatedFiles []string `json:"updated_files"`
}

const defaultGradusYAML = `schema: gradus/v1

levels:
  - id: project
    source: { path: .gradus/levels/project }

  - id: user
    source: { path: ~/.gradus/levels/user }

targets: [claude-code, antigravity, agents-md]

hooks:
  manager: auto

evidence:
  sinks:
    - type: local
`

const defaultProjectLevelYAML = `schema: gradus/v1
id: project
charter:
  purpose: "Repository-specific rules and conventions."
  allowed_severities: [MUST, SHOULD, MAY]
`

const defaultExampleRuleMD = `---
id: PRJ-GEN-001
title: Keep changes scoped to the requested task
severity: SHOULD
enforcement: prompt
scope: ["**/*"]
source: "Gradus project starter template"
tags: [general, scope]
version: 1
---
Keep code and documentation changes focused on the requested task without unrelated modifications.

## Why
Focused changes are easier to review, verify, and revert safely.
`

var gradusGitignoreEntries = []string{
	".gradus/cache/",
	".gradus/evidence/",
	"CLAUDE.local.md",
	".agents/rules/gradus-user-*.md",
}

func newInitCmd(jsonOutput *bool) *cobra.Command {
	var force bool
	var dir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a Gradus pyramid in the current repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			res, err := runInit(dir, force)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}

			for _, f := range res.CreatedFiles {
				fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", f)
			}
			for _, f := range res.UpdatedFiles {
				fmt.Fprintf(cmd.OutOrStdout(), "updated %s\n", f)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing gradus.yaml if present")
	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}

func runInit(rootDir string, force bool) (initResult, error) {
	var res initResult

	gradusPath := filepath.Join(rootDir, "gradus.yaml")
	if _, err := os.Stat(gradusPath); err == nil && !force {
		return res, fmt.Errorf("gradus.yaml already exists (use --force to overwrite)")
	} else if err != nil && !os.IsNotExist(err) {
		return res, fmt.Errorf("failed to inspect %s: %w", gradusPath, err)
	}

	rulesDir := filepath.Join(rootDir, ".gradus", "levels", "project", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return res, fmt.Errorf("failed to create %s: %w", rulesDir, err)
	}

	if err := os.WriteFile(gradusPath, []byte(defaultGradusYAML), 0o644); err != nil {
		return res, fmt.Errorf("failed to write gradus.yaml: %w", err)
	}
	res.CreatedFiles = append(res.CreatedFiles, "gradus.yaml")

	levelPath := filepath.Join(rootDir, ".gradus", "levels", "project", "level.yaml")
	if err := os.WriteFile(levelPath, []byte(defaultProjectLevelYAML), 0o644); err != nil {
		return res, fmt.Errorf("failed to write level.yaml: %w", err)
	}
	res.CreatedFiles = append(res.CreatedFiles, filepath.ToSlash(filepath.Join(".gradus", "levels", "project", "level.yaml")))

	rulePath := filepath.Join(rulesDir, "PRJ-GEN-001.md")
	if err := os.WriteFile(rulePath, []byte(defaultExampleRuleMD), 0o644); err != nil {
		return res, fmt.Errorf("failed to write example rule: %w", err)
	}
	res.CreatedFiles = append(res.CreatedFiles, filepath.ToSlash(filepath.Join(".gradus", "levels", "project", "rules", "PRJ-GEN-001.md")))

	gitignorePath := filepath.Join(rootDir, ".gitignore")
	createdGitignore, updatedGitignore, err := ensureGitignoreEntries(gitignorePath, gradusGitignoreEntries)
	if err != nil {
		return res, err
	}
	if createdGitignore {
		res.CreatedFiles = append(res.CreatedFiles, ".gitignore")
	} else if updatedGitignore {
		res.UpdatedFiles = append(res.UpdatedFiles, ".gitignore")
	}

	return res, nil
}

func ensureGitignoreEntries(path string, entries []string) (created bool, updated bool, err error) {
	existingBytes, readErr := os.ReadFile(path)
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, false, fmt.Errorf("failed to read .gitignore: %w", readErr)
	}

	existed := readErr == nil
	content := strings.ReplaceAll(string(existingBytes), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	present := make(map[string]bool, len(lines))
	for _, line := range lines {
		present[strings.TrimSpace(line)] = true
	}

	var missing []string
	for _, entry := range entries {
		if !present[entry] {
			missing = append(missing, entry)
		}
	}

	if len(missing) == 0 && existed {
		return false, false, nil
	}

	var b strings.Builder
	b.WriteString(content)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		b.WriteString("\n")
	}
	if len(missing) > 0 {
		if len(content) > 0 {
			b.WriteString("\n")
		}
		b.WriteString("# Gradus\n")
		for _, m := range missing {
			b.WriteString(m)
			b.WriteString("\n")
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return false, false, fmt.Errorf("failed to write .gitignore: %w", err)
	}
	if !existed {
		return true, false, nil
	}
	return false, true, nil
}
