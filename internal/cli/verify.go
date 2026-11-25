package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/skills"
)

// DriftItem describes a divergence between expected compiled output and disk state.
type DriftItem struct {
	Path   string `json:"path"`
	Reason string `json:"reason"` // missing | modified | stale
}

type verifyReport struct {
	Verdict           string             `json:"verdict"`
	ViewHash          string             `json:"view_hash"`
	LockHash          string             `json:"lock_hash"`
	Conflicts         []resolve.Conflict `json:"conflicts"`
	Drift             []DriftItem        `json:"drift"`
	ExpiredExceptions []string           `json:"expired_exceptions,omitempty"`
	Warnings          []string           `json:"warnings,omitempty"`
	Notes             []string           `json:"notes,omitempty"`
	EvidenceWritten   []string           `json:"evidence_written,omitempty"`
}

func newVerifyCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var writeEvidence, checkModelTier bool

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify pyramid integrity, constraints, conflicts, expired exceptions, and generated-file drift",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			rep, exitCode, err := runVerify(dir, writeEvidence, checkModelTier)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else {
				out := cmd.OutOrStdout()
				if exitCode == ExitPass {
					fmt.Fprintf(out, "verify passed (view %s)\n", rep.ViewHash)
				} else {
					for _, c := range rep.Conflicts {
						fmt.Fprintf(out, "CONFLICT [%s]: %s\n", c.Kind, c.Message)
					}
					for _, d := range rep.Drift {
						fmt.Fprintf(out, "DRIFT [%s]: %s\n", d.Reason, d.Path)
					}
					for _, ex := range rep.ExpiredExceptions {
						fmt.Fprintf(out, "EXPIRED EXCEPTION: %s\n", ex)
					}
				}
				for _, w := range rep.Warnings {
					fmt.Fprintf(out, "WARNING: %s\n", w)
				}
			}

			if exitCode != ExitPass {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: exitCode}
				}
				return &ExitErrorWithCode{
					Code: exitCode,
					Err:  fmt.Errorf("verification failed: %d conflict(s), %d drift item(s), %d expired exception(s)", len(rep.Conflicts), len(rep.Drift), len(rep.ExpiredExceptions)),
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().BoolVar(&writeEvidence, "write-evidence", false, "Persist verification evidence to configured sinks")
	cmd.Flags().BoolVar(&checkModelTier, "check-model-tier", false, "Verify local evidence records meet declared models.operations capability tiers")
	return cmd
}

func runVerify(dir string, writeEvidence, checkModelTier bool) (verifyReport, int, error) {
	var rep verifyReport

	cfgBytes, err := os.ReadFile(filepath.Join(dir, "gradus.yaml"))
	if err != nil {
		return rep, ExitError, fmt.Errorf("failed to read gradus.yaml: %w", err)
	}
	cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
	if err != nil {
		return rep, ExitError, err
	}

	resolved, err := resolve.Resolve(dir, resolve.Options{})
	if err != nil {
		return rep, ExitError, err
	}

	plan, err := compile.PlanCompile(dir, resolved.View, cfgRes.Value.Targets, skills.FS)
	if err != nil {
		return rep, ExitError, err
	}

	drift := detectCompileDrift(dir, plan, cfgRes.Value.Targets)

	var expired []string
	var otherNotes []string
	for _, n := range resolved.View.Notes {
		if strings.HasPrefix(n, "exception ") && strings.Contains(n, " expired on ") {
			expired = append(expired, n)
		} else {
			otherNotes = append(otherNotes, n)
		}
	}

	conflicts := resolved.View.Conflicts
	if checkModelTier && resolved.View.Models != nil {
		tierConflicts, err := verifyEvidenceModelTiers(dir, resolved.View.Models)
		if err != nil {
			return rep, ExitError, err
		}
		conflicts = append(conflicts, tierConflicts...)
	}

	verdict := "pass"
	exitCode := ExitPass
	if len(conflicts) > 0 || len(drift) > 0 || len(expired) > 0 {
		verdict = "fail"
		exitCode = ExitViolation
	}

	rep = verifyReport{
		Verdict:           verdict,
		ViewHash:          resolved.View.ViewHash,
		LockHash:          resolved.View.LockHash,
		Conflicts:         conflicts,
		Drift:             drift,
		ExpiredExceptions: expired,
		Warnings:          resolved.View.Warnings,
		Notes:             otherNotes,
	}

	if writeEvidence {
		var findings []evidence.Finding
		for _, c := range resolved.View.Conflicts {
			findings = append(findings, evidence.Finding{
				RuleID:     c.RuleID,
				Severity:   "MUST",
				Level:      c.Level,
				Message:    c.Message,
				Confidence: 1.0,
				Status:     "open",
			})
		}
		for _, d := range drift {
			findings = append(findings, evidence.Finding{
				RuleID:     "GRADUS-DRIFT",
				Severity:   "MUST",
				Path:       d.Path,
				Message:    fmt.Sprintf("generated file %s is %s", d.Path, d.Reason),
				Confidence: 1.0,
				Status:     "open",
			})
		}
		rec := evidence.NewRecord(dir, "verify", verdict, "", resolved.View.LockHash, resolved.View.ViewHash, findings, time.Now().UTC())
		written, err := evidence.WriteRecord(dir, cfgRes.Value.Evidence, rec, time.Now().UTC())
		if err != nil {
			return rep, ExitError, err
		}
		rep.EvidenceWritten = written
	}

	return rep, exitCode, nil
}

func verifyEvidenceModelTiers(dir string, mp *resolve.ResolvedModelPolicy) ([]resolve.Conflict, error) {
	if mp == nil || len(mp.Operations) == 0 {
		return nil, nil
	}
	records, err := evidence.ListLocalRecords(dir)
	if err != nil {
		return nil, err
	}
	ops := make([]string, 0, len(mp.Operations))
	for op := range mp.Operations {
		ops = append(ops, op)
	}
	sort.Strings(ops)

	var out []resolve.Conflict
	for _, op := range ops {
		reqTier := mp.Operations[op]
		var latest *evidence.Record
		for i := range records {
			r := &records[i]
			skillOp := ""
			if r.Producer.Skill != nil {
				skillOp = strings.TrimPrefix(strings.TrimSpace(*r.Producer.Skill), "gradus-")
			}
			if skillOp == op || r.Kind == op || (op == "conflicts" && r.Kind == "semantic-conflicts") {
				latest = r
				break
			}
		}
		if latest == nil {
			out = append(out, resolve.Conflict{
				Kind:    "model_policy",
				Target:  op,
				Message: fmt.Sprintf("missing local evidence record for governed operation %q (requires %s)", op, reqTier),
			})
			continue
		}
		actualTier := ""
		if latest.Producer.ModelTier != nil {
			actualTier = strings.TrimSpace(*latest.Producer.ModelTier)
		}
		if actualTier == "" {
			out = append(out, resolve.Conflict{
				Kind:    "model_policy",
				Target:  op,
				Message: fmt.Sprintf("latest evidence %s for operation %q has no recorded model_tier (requires %s)", latest.ID, op, reqTier),
			})
			continue
		}
		if resolve.TierRank(actualTier) < resolve.TierRank(reqTier) {
			out = append(out, resolve.Conflict{
				Kind:    "model_policy",
				Target:  op,
				Message: fmt.Sprintf("latest evidence %s for operation %q used %s, below required %s", latest.ID, op, actualTier, reqTier),
			})
		}
	}
	return out, nil
}

func detectCompileDrift(rootDir string, plan map[string][]byte, targets []string) []DriftItem {
	drift := []DriftItem{}

	paths := make([]string, 0, len(plan))
	for p := range plan {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	claudeKeep := make(map[string]bool)
	agentsKeep := make(map[string]bool)

	for _, rel := range paths {
		if strings.HasPrefix(rel, ".claude/rules/") {
			claudeKeep[filepath.Base(rel)] = true
		}
		if strings.HasPrefix(rel, ".agents/rules/") {
			agentsKeep[filepath.Base(rel)] = true
		}

		abs := filepath.Join(rootDir, filepath.FromSlash(rel))
		diskBytes, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				drift = append(drift, DriftItem{Path: rel, Reason: "missing"})
			} else {
				drift = append(drift, DriftItem{Path: rel, Reason: "unreadable"})
			}
			continue
		}
		if !bytes.Equal(diskBytes, plan[rel]) {
			drift = append(drift, DriftItem{Path: rel, Reason: "modified"})
		}
	}

	for _, target := range targets {
		switch strings.TrimSpace(strings.ToLower(target)) {
		case "claude-code":
			drift = append(drift, findStaleFiles(filepath.Join(rootDir, ".claude", "rules"), ".claude/rules/", claudeKeep)...)
		case "antigravity":
			drift = append(drift, findStaleFiles(filepath.Join(rootDir, ".agents", "rules"), ".agents/rules/", agentsKeep)...)
		}
	}

	return drift
}

func findStaleFiles(dir, relPrefix string, keep map[string]bool) []DriftItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []DriftItem
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "gradus-") && strings.HasSuffix(name, ".md") && !keep[name] {
			out = append(out, DriftItem{Path: relPrefix + name, Reason: "stale"})
		}
	}
	return out
}
