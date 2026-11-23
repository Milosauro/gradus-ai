package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/state"
)

func newExceptionCmd(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exception",
		Short: "Request, approve, list, expire exceptions, or record break-glass waivers",
	}

	cmd.AddCommand(newExceptionRequestCmd(jsonOutput))
	cmd.AddCommand(newExceptionApproveCmd(jsonOutput))
	cmd.AddCommand(newExceptionBreakGlassCmd(jsonOutput))
	cmd.AddCommand(newExceptionListCmd(jsonOutput))
	cmd.AddCommand(newExceptionExpireCmd(jsonOutput))
	return cmd
}

func newExceptionRequestCmd(jsonOutput *bool) *cobra.Command {
	var dir, ruleID, reason, expires string
	var scope []string

	cmd := &cobra.Command{
		Use:   "request",
		Short: "Create a pending exception request under .gradus/exceptions/",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(expires)); err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("invalid --expires date %q (expected YYYY-MM-DD)", expires)}
			}
			now := time.Now().UTC()
			rec := state.ExceptionRecord{
				Schema:     schema.SchemaVersion,
				ID:         state.NewExceptionID("exception", now),
				RuleID:     ruleID,
				Scope:      scope,
				Reason:     reason,
				ApprovedBy: []string{},
				Created:    now.Format("2006-01-02"),
				Expires:    expires,
				Kind:       "exception",
			}
			path, err := state.WriteException(dir, rec)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rec)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created pending exception %s for rule %s (%s)\n", rec.ID, rec.RuleID, path)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&ruleID, "rule", "", "Target rule ID (required)")
	cmd.Flags().StringVar(&reason, "reason", "", "Justification for the exception (required)")
	cmd.Flags().StringVar(&expires, "expires", "", "Expiration date YYYY-MM-DD (required)")
	cmd.Flags().StringSliceVar(&scope, "scope", []string{"**/*"}, "Glob scopes covered by the exception")
	_ = cmd.MarkFlagRequired("rule")
	_ = cmd.MarkFlagRequired("reason")
	_ = cmd.MarkFlagRequired("expires")
	return cmd
}

func newExceptionApproveCmd(jsonOutput *bool) *cobra.Command {
	var dir, approver string

	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a pending exception by recording an approver",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			if err := validateExceptionApprover(dir, args[0], approver); err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			rec, err := state.ApproveException(dir, args[0], approver)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rec)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Approved exception %s (approvers: %s)\n", rec.ID, strings.Join(rec.ApprovedBy, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&approver, "by", "", "Approver identifier (required)")
	_ = cmd.MarkFlagRequired("by")
	return cmd
}

func newExceptionBreakGlassCmd(jsonOutput *bool) *cobra.Command {
	var dir, ruleID, reason, expires, postReviewDue, actor string
	var scope []string

	cmd := &cobra.Command{
		Use:   "break-glass",
		Short: "Record an emergency break-glass waiver when permitted by the target level policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(expires)); err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("invalid --expires date %q (expected YYYY-MM-DD)", expires)}
			}
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(postReviewDue)); err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("invalid --post-review-due date %q (expected YYYY-MM-DD)", postReviewDue)}
			}
			pyramid, err := schema.LoadLocalPyramid(dir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			var targetLevel string
			for _, r := range pyramid.Rules {
				if r.ID == ruleID {
					targetLevel = r.Level
					break
				}
			}
			if targetLevel == "" {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("rule %q not found in local pyramid", ruleID)}
			}

			now := time.Now().UTC()
			rec := state.ExceptionRecord{
				Schema:        schema.SchemaVersion,
				ID:            state.NewExceptionID("break-glass", now),
				RuleID:        ruleID,
				Scope:         scope,
				Reason:        reason,
				ApprovedBy:    []string{actor},
				Created:       now.Format("2006-01-02"),
				Expires:       expires,
				Kind:          "break-glass",
				PostReviewDue: postReviewDue,
			}

			var lcPtr *schema.LevelConfig
			if lc, ok := pyramid.Levels[targetLevel]; ok {
				lcPtr = &lc
			}
			var offlinePtr *schema.OfflinePolicy
			for _, l := range pyramid.Config.Levels {
				if l.ID == targetLevel {
					offlinePtr = l.Offline
					break
				}
			}

			policyConflicts := resolve.ValidateExceptionAgainstLevel(resolve.ExceptionView{
				ID:            rec.ID,
				RuleID:        rec.RuleID,
				Scope:         rec.Scope,
				Kind:          rec.Kind,
				Reason:        rec.Reason,
				Created:       rec.Created,
				Expires:       rec.Expires,
				PostReviewDue: rec.PostReviewDue,
				ApprovedBy:    rec.ApprovedBy,
			}, targetLevel, lcPtr, offlinePtr, now)
			if len(policyConflicts) > 0 {
				if *jsonOutput {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					_ = enc.Encode(map[string]any{"verdict": "fail", "conflicts": policyConflicts})
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{Code: ExitViolation, Err: fmt.Errorf("%s", policyConflicts[0].Message)}
			}

			path, err := state.WriteException(dir, rec)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rec)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Recorded break-glass %s for rule %s (%s)\n", rec.ID, rec.RuleID, path)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&ruleID, "rule", "", "Target rule ID (required)")
	cmd.Flags().StringVar(&reason, "reason", "", "Incident justification (required)")
	cmd.Flags().StringVar(&expires, "expires", "", "Expiration date YYYY-MM-DD (required)")
	cmd.Flags().StringVar(&postReviewDue, "post-review-due", "", "Mandatory post-review date YYYY-MM-DD (required)")
	cmd.Flags().StringVar(&actor, "by", "", "Responder recording break-glass (required)")
	cmd.Flags().StringSliceVar(&scope, "scope", []string{"**/*"}, "Glob scopes covered by break-glass")
	_ = cmd.MarkFlagRequired("rule")
	_ = cmd.MarkFlagRequired("reason")
	_ = cmd.MarkFlagRequired("expires")
	_ = cmd.MarkFlagRequired("post-review-due")
	_ = cmd.MarkFlagRequired("by")
	return cmd
}

func newExceptionListCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all exceptions and their computed lifecycle status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			items, err := state.ListExceptions(dir, time.Now().UTC())
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{"exceptions": items})
			}
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No exceptions recorded.")
				return nil
			}
			for _, ex := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] (%s) rule=%s expires=%s\n", ex.ID, ex.Status, ex.Kind, ex.RuleID, ex.Expires)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}

func newExceptionExpireCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "expire <id>",
		Short: "Immediately expire an exception by setting its expiration date in the past",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			rec, err := state.ExpireException(dir, args[0], time.Now().UTC())
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rec)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Expired exception %s (expires=%s)\n", rec.ID, rec.Expires)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}

func validateExceptionApprover(rootDir, exceptionID, approver string) error {
	trimmed := strings.TrimSpace(approver)
	if trimmed == "" {
		return fmt.Errorf("--by approver cannot be empty")
	}

	exceptions, err := state.ListExceptions(rootDir, time.Now().UTC())
	if err != nil {
		return err
	}
	var ruleID string
	for _, ex := range exceptions {
		if ex.ID == exceptionID {
			ruleID = ex.RuleID
			break
		}
	}
	if ruleID == "" {
		return nil // Let state.ApproveException return the canonical not-found error.
	}

	res, err := resolve.Resolve(rootDir, resolve.Options{OfflineOnly: true, IncludeDraft: true})
	if err != nil {
		return err
	}
	var targetLevel string
	for _, r := range append(res.View.Rules, res.View.InactiveRules...) {
		if r.ID == ruleID {
			targetLevel = r.Level
			break
		}
	}
	if targetLevel == "" {
		return nil
	}

	cfgBytes, err := os.ReadFile(filepath.Join(rootDir, "gradus.yaml"))
	if err != nil {
		return err
	}
	cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
	if err != nil {
		return err
	}

	allowed := make(map[string]bool)
	for _, lvl := range cfgRes.Value.Levels {
		if lvl.ID != targetLevel {
			continue
		}
		for _, a := range lvl.Approvers {
			allowed[strings.TrimSpace(a)] = true
		}
		var lvlDir string
		if lvl.Source.Path != "" {
			if filepath.IsAbs(lvl.Source.Path) {
				lvlDir = lvl.Source.Path
			} else {
				lvlDir = filepath.Join(rootDir, filepath.FromSlash(lvl.Source.Path))
			}
		} else if lvl.Source.Git != "" {
			lvlDir = filepath.Join(rootDir, ".gradus", "cache", lvl.ID, "repo")
		}
		if lvlDir != "" {
			if lvlBytes, readErr := os.ReadFile(filepath.Join(lvlDir, "level.yaml")); readErr == nil {
				if lcRes, parseErr := schema.ParseLevelConfig(lvlBytes); parseErr == nil {
					for _, a := range lcRes.Value.Approvers {
						allowed[strings.TrimSpace(a)] = true
					}
					for _, o := range lcRes.Value.Owners {
						allowed[strings.TrimSpace(o)] = true
					}
					if lcRes.Value.Exceptions != nil {
						for _, a := range lcRes.Value.Exceptions.Approvers {
							allowed[strings.TrimSpace(a)] = true
						}
					}
				}
			}
		}
	}
	delete(allowed, "")

	if len(allowed) > 0 && !allowed[trimmed] {
		return fmt.Errorf("approver %q is not authorized to approve exceptions for level %q", approver, targetLevel)
	}
	return nil
}
