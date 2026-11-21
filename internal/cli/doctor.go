package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pass | warn | fail
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

type doctorReport struct {
	Verdict string        `json:"verdict"` // pass | fail
	Checks  []doctorCheck `json:"checks"`
}

func newDoctorCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose Gradus configuration, lockfile, PATH, and agent hook files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			rep := runDoctor(dir)

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else {
				out := cmd.OutOrStdout()
				for _, c := range rep.Checks {
					fmt.Fprintf(out, "[%s] %s: %s\n", strings.ToUpper(c.Status), c.Name, c.Message)
					if c.Fix != "" {
						fmt.Fprintf(out, "       fix: %s\n", c.Fix)
					}
				}
			}

			if rep.Verdict == "fail" {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{Code: ExitViolation, Err: fmt.Errorf("doctor found failing diagnostics")}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}

func runDoctor(dir string) doctorReport {
	var checks []doctorCheck
	var cfg *schema.PyramidConfig

	// 1. gradus.yaml
	cfgPath := filepath.Join(dir, "gradus.yaml")
	if raw, err := os.ReadFile(cfgPath); err != nil {
		checks = append(checks, doctorCheck{
			Name:    "gradus_yaml",
			Status:  "fail",
			Message: fmt.Sprintf("cannot read gradus.yaml: %v", err),
			Fix:     "Run `gradusctl init` to create gradus.yaml.",
		})
	} else if parsed, perr := schema.ParsePyramidConfig(raw); perr != nil {
		checks = append(checks, doctorCheck{
			Name:    "gradus_yaml",
			Status:  "fail",
			Message: fmt.Sprintf("invalid gradus.yaml: %v", perr),
			Fix:     "Fix the schema error in gradus.yaml and re-run `gradusctl doctor`.",
		})
	} else {
		cfg = &parsed.Value
		checks = append(checks, doctorCheck{
			Name:    "gradus_yaml",
			Status:  "pass",
			Message: "gradus.yaml is valid",
		})
	}

	// 2. gradus.lock
	lockPath := filepath.Join(dir, "gradus.lock")
	if _, err := lock.ReadLockfile(lockPath); err != nil {
		checks = append(checks, doctorCheck{
			Name:    "gradus_lock",
			Status:  "fail",
			Message: fmt.Sprintf("invalid or missing gradus.lock: %v", err),
			Fix:     "Run `gradusctl resolve --write` to generate gradus.lock.",
		})
	} else {
		checks = append(checks, doctorCheck{
			Name:    "gradus_lock",
			Status:  "pass",
			Message: "gradus.lock is valid",
		})
	}

	// 3. gradusctl on PATH
	if path, err := exec.LookPath("gradusctl"); err != nil {
		checks = append(checks, doctorCheck{
			Name:    "gradusctl_path",
			Status:  "warn",
			Message: "gradusctl binary not found on PATH (agent hooks invoke `gradusctl` by name)",
			Fix:     "Ensure the directory containing gradusctl (e.g. $HOME/go/bin) is on your PATH.",
		})
	} else {
		checks = append(checks, doctorCheck{
			Name:    "gradusctl_path",
			Status:  "pass",
			Message: fmt.Sprintf("gradusctl found at %s", path),
		})
	}

	// 4. .claude/settings.json
	claudePath := filepath.Join(dir, ".claude", "settings.json")
	_, claudeStatErr := os.Stat(claudePath)
	if (cfg != nil && slices.Contains(cfg.Targets, "claude-code")) || claudeStatErr == nil {
		checks = append(checks, checkClaudeSettings(claudePath))
	}

	// 5. .agents/hooks.json
	agPath := filepath.Join(dir, ".agents", "hooks.json")
	_, agStatErr := os.Stat(agPath)
	if (cfg != nil && slices.Contains(cfg.Targets, "antigravity")) || agStatErr == nil {
		checks = append(checks, checkAntigravityHooks(agPath))
	}

	verdict := "pass"
	for _, c := range checks {
		if c.Status == "fail" {
			verdict = "fail"
			break
		}
	}
	return doctorReport{Verdict: verdict, Checks: checks}
}

func checkClaudeSettings(path string) doctorCheck {
	raw, err := os.ReadFile(path)
	if err != nil {
		return doctorCheck{
			Name:    "claude_settings",
			Status:  "fail",
			Message: fmt.Sprintf("cannot read .claude/settings.json: %v", err),
			Fix:     "Run `gradusctl compile` to generate .claude/settings.json.",
		}
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return doctorCheck{
			Name:    "claude_settings",
			Status:  "fail",
			Message: fmt.Sprintf("malformed JSON in .claude/settings.json: %v", err),
			Fix:     "Remove .claude/settings.json and re-run `gradusctl compile`.",
		}
	}

	validTypes := map[string]bool{"command": true, "prompt": true, "agent": true, "http": true, "mcp_tool": true}
	hooksMap, _ := doc["hooks"].(map[string]any)
	hasGuard := false

	for eventName, rawEntries := range hooksMap {
		entries, ok := rawEntries.([]any)
		if !ok {
			continue
		}
		for _, rawEntry := range entries {
			entry, ok := rawEntry.(map[string]any)
			if !ok {
				continue
			}
			innerHooks, ok := entry["hooks"].([]any)
			if !ok {
				continue
			}
			for _, rawHook := range innerHooks {
				hm, ok := rawHook.(map[string]any)
				if !ok {
					continue
				}
				htype, _ := hm["type"].(string)
				if !validTypes[htype] {
					return doctorCheck{
						Name:    "claude_settings",
						Status:  "fail",
						Message: fmt.Sprintf(".claude/settings.json hook under %q has invalid type %q (valid: command, prompt, agent, http, mcp_tool)", eventName, htype),
						Fix:     "Remove the invalid hook entry or delete .claude/settings.json and re-run `gradusctl compile`.",
					}
				}
				if eventName == "PreToolUse" {
					if cmd, _ := hm["command"].(string); strings.Contains(cmd, "gradusctl guard") {
						hasGuard = true
					}
				}
			}
		}
	}

	if !hasGuard {
		return doctorCheck{
			Name:    "claude_settings",
			Status:  "fail",
			Message: ".claude/settings.json is missing a PreToolUse `gradusctl guard` hook",
			Fix:     "Run `gradusctl compile` and restart your Claude Code session.",
		}
	}
	return doctorCheck{
		Name:    "claude_settings",
		Status:  "pass",
		Message: ".claude/settings.json hook schema and PreToolUse guard are valid",
	}
}

func checkAntigravityHooks(path string) doctorCheck {
	raw, err := os.ReadFile(path)
	if err != nil {
		return doctorCheck{
			Name:    "antigravity_hooks",
			Status:  "fail",
			Message: fmt.Sprintf("cannot read .agents/hooks.json: %v", err),
			Fix:     "Run `gradusctl compile` to generate .agents/hooks.json.",
		}
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return doctorCheck{
			Name:    "antigravity_hooks",
			Status:  "fail",
			Message: fmt.Sprintf("malformed JSON in .agents/hooks.json: %v", err),
			Fix:     "Remove .agents/hooks.json and re-run `gradusctl compile`.",
		}
	}
	if !strings.Contains(string(raw), "gradusctl guard") {
		return doctorCheck{
			Name:    "antigravity_hooks",
			Status:  "fail",
			Message: ".agents/hooks.json is missing a `gradusctl guard` PreToolUse hook",
			Fix:     "Run `gradusctl compile` and start a new Antigravity session.",
		}
	}
	return doctorCheck{
		Name:    "antigravity_hooks",
		Status:  "pass",
		Message: ".agents/hooks.json is valid and registers `gradusctl guard`",
	}
}
