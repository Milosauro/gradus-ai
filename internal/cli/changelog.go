package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/changelog"
	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func newChangelogCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var fromRef string
	var toRef string

	cmd := &cobra.Command{
		Use:   "changelog <level-id>",
		Short: "Show a readable diff of rule and metadata changes between two versions of a git-backed level",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			levelID := args[0]

			rep, err := runChangelog(dir, levelID, fromRef, toRef)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				return nil
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), changelog.FormatMarkdown(rep))
			return err
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&fromRef, "from", "", "Starting git ref or commit (defaults to locked commit in gradus.lock)")
	cmd.Flags().StringVar(&toRef, "to", "", "Target git ref or commit (defaults to source.ref in gradus.yaml or HEAD)")
	return cmd
}

func runChangelog(rootDir, levelID, fromRef, toRef string) (changelog.Report, error) {
	var rep changelog.Report

	cfgBytes, err := os.ReadFile(filepath.Join(rootDir, "gradus.yaml"))
	if err != nil {
		return rep, fmt.Errorf("failed to read gradus.yaml: %w", err)
	}
	cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
	if err != nil {
		return rep, err
	}

	var targetLvl *schema.LevelRef
	for i := range cfgRes.Value.Levels {
		if cfgRes.Value.Levels[i].ID == levelID {
			targetLvl = &cfgRes.Value.Levels[i]
			break
		}
	}
	if targetLvl == nil {
		return rep, fmt.Errorf("level %q not found in gradus.yaml", levelID)
	}
	if strings.TrimSpace(targetLvl.Source.Git) == "" {
		return rep, fmt.Errorf("level %q is not git-backed; changelog requires a git source", levelID)
	}

	resolvedFrom := strings.TrimSpace(fromRef)
	if resolvedFrom == "" {
		lf, lockErr := lock.ReadLockfile(filepath.Join(rootDir, "gradus.lock"))
		if lockErr != nil {
			return rep, fmt.Errorf("--from omitted and failed to read gradus.lock: %w", lockErr)
		}
		entry, ok := lf.FindEntry("level", levelID)
		if !ok || strings.TrimSpace(entry.Commit) == "" {
			return rep, fmt.Errorf("--from omitted and no locked commit found for level %q in gradus.lock", levelID)
		}
		resolvedFrom = entry.Commit
	}

	resolvedTo := strings.TrimSpace(toRef)
	if resolvedTo == "" {
		resolvedTo = strings.TrimSpace(targetLvl.Source.Ref)
		if resolvedTo == "" {
			resolvedTo = "HEAD"
		}
	}

	if strings.HasPrefix(resolvedFrom, "-") {
		return rep, fmt.Errorf("invalid --from ref %q (cannot start with '-')", resolvedFrom)
	}
	if strings.HasPrefix(resolvedTo, "-") {
		return rep, fmt.Errorf("invalid --to ref %q (cannot start with '-')", resolvedTo)
	}

	cacheRepoDir := filepath.Join(rootDir, ".gradus", "cache", levelID, "repo")
	repoDir := cacheRepoDir
	if _, statErr := os.Stat(filepath.Join(cacheRepoDir, ".git")); statErr == nil {
		// Fetch missing objects/tags without touching the cached checkout or HEAD (NFR3).
		if !hasGitRef(cacheRepoDir, resolvedFrom) || !hasGitRef(cacheRepoDir, resolvedTo) {
			if err := runGitChangelog(cacheRepoDir, "fetch", "--tags", "--force", "origin"); err != nil {
				return rep, fmt.Errorf("failed to fetch level %q git source: %w", levelID, err)
			}
		}
	} else {
		tmpRepo, err := os.MkdirTemp("", "gradus-changelog-repo-*")
		if err != nil {
			return rep, fmt.Errorf("failed to create temp git repo: %w", err)
		}
		defer os.RemoveAll(tmpRepo)
		if err := runGitChangelog("", "clone", "--no-checkout", "--", targetLvl.Source.Git, tmpRepo); err != nil {
			return rep, fmt.Errorf("failed to clone level %q git source: %w", levelID, err)
		}
		repoDir = tmpRepo
	}

	return changelog.DiffLevel(repoDir, levelID, resolvedFrom, resolvedTo)
}

func runGitChangelog(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func hasGitRef(repoDir, ref string) bool {
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		return false
	}
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd.Run() == nil
}
