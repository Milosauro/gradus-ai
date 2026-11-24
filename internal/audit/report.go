package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

// Options controls audit bundle construction.
type Options struct {
	Framework string // Optional case-insensitive framework filter (e.g. "ISO27001")
	Now       time.Time
}

// RuleRef summarizes a rule covering a compliance control.
type RuleRef struct {
	RuleID      string `json:"rule_id"`
	Title       string `json:"title"`
	Level       string `json:"level"`
	Severity    string `json:"severity"`
	Enforcement string `json:"enforcement"`
	Waived      bool   `json:"waived,omitempty"`
	Source      string `json:"source"`
}

// ControlCoverage lists all rules mapped to a single framework control ID.
type ControlCoverage struct {
	Control string    `json:"control"`
	Rules   []RuleRef `json:"rules"`
}

// FrameworkCoverage aggregates control mappings for one compliance framework.
type FrameworkCoverage struct {
	Framework    string            `json:"framework"`
	ControlCount int               `json:"control_count"`
	RuleCount    int               `json:"rule_count"`
	Controls     []ControlCoverage `json:"controls"`
}

// AuditBundle is a self-contained, process-only audit export (X-02, P16).
type AuditBundle struct {
	Schema        string                  `json:"schema"`
	GeneratedAt   string                  `json:"generated_at"`
	Repository    string                  `json:"repository"`
	ViewHash      string                  `json:"view_hash"`
	LockHash      string                  `json:"lock_hash,omitempty"`
	Profile       *schema.Profile         `json:"profile,omitempty"`
	Levels        []resolve.LevelSummary  `json:"levels"`
	Overlays      []resolve.OverlaySummary `json:"overlays,omitempty"`
	Conflicts     []resolve.Conflict      `json:"conflicts"`
	Frameworks    []FrameworkCoverage     `json:"frameworks"`
	UnmappedRules []string                `json:"unmapped_rules,omitempty"`
	Exceptions    []resolve.ExceptionView `json:"exceptions"`
	Evidence      []evidence.Record       `json:"evidence,omitempty"`
}

// SplitControl splits "FRAMEWORK:CONTROL" into ("FRAMEWORK", "CONTROL").
// Unprefixed identifiers map to framework "UNSPECIFIED".
func SplitControl(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		fw := strings.TrimSpace(trimmed[:idx])
		ctrl := strings.TrimSpace(trimmed[idx+1:])
		if fw != "" && ctrl != "" {
			return fw, ctrl
		}
	}
	return "UNSPECIFIED", trimmed
}

// BuildBundle constructs a deterministic AuditBundle from a ResolvedView and local evidence records.
func BuildBundle(rootDir string, view resolve.ResolvedView, opts Options) (AuditBundle, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	// framework -> control -> []RuleRef
	fwMap := make(map[string]map[string][]RuleRef)
	fwRuleSet := make(map[string]map[string]bool)
	var unmapped []string

	for _, r := range view.Rules {
		if len(r.Controls) == 0 {
			unmapped = append(unmapped, r.ID)
			continue
		}
		ref := RuleRef{
			RuleID:      r.ID,
			Title:       r.Title,
			Level:       r.Level,
			Severity:    r.Severity,
			Enforcement: r.Enforcement,
			Waived:      r.Waived,
			Source:      r.Provenance.Source,
		}
		for _, rawCtrl := range r.Controls {
			fw, ctrl := SplitControl(rawCtrl)
			if ctrl == "" {
				continue
			}
			if opts.Framework != "" && !strings.EqualFold(fw, strings.TrimSpace(opts.Framework)) {
				continue
			}
			if fwMap[fw] == nil {
				fwMap[fw] = make(map[string][]RuleRef)
				fwRuleSet[fw] = make(map[string]bool)
			}
			fwMap[fw][ctrl] = append(fwMap[fw][ctrl], ref)
			fwRuleSet[fw][r.ID] = true
		}
	}
	sort.Strings(unmapped)

	fwNames := make([]string, 0, len(fwMap))
	for fw := range fwMap {
		fwNames = append(fwNames, fw)
	}
	sort.Strings(fwNames)

	frameworks := make([]FrameworkCoverage, 0, len(fwNames))
	for _, fw := range fwNames {
		ctrlMap := fwMap[fw]
		ctrlNames := make([]string, 0, len(ctrlMap))
		for c := range ctrlMap {
			ctrlNames = append(ctrlNames, c)
		}
		sort.Strings(ctrlNames)

		controls := make([]ControlCoverage, 0, len(ctrlNames))
		for _, c := range ctrlNames {
			rules := ctrlMap[c]
			sort.Slice(rules, func(i, j int) bool { return rules[i].RuleID < rules[j].RuleID })
			controls = append(controls, ControlCoverage{
				Control: c,
				Rules:   rules,
			})
		}
		frameworks = append(frameworks, FrameworkCoverage{
			Framework:    fw,
			ControlCount: len(controls),
			RuleCount:    len(fwRuleSet[fw]),
			Controls:     controls,
		})
	}

	evRecords := loadLocalEvidence(rootDir)

	conflicts := view.Conflicts
	if conflicts == nil {
		conflicts = []resolve.Conflict{}
	}
	exceptions := view.Exceptions
	if exceptions == nil {
		exceptions = []resolve.ExceptionView{}
	}

	return AuditBundle{
		Schema:        schema.SchemaVersion,
		GeneratedAt:   now.UTC().Format(time.RFC3339),
		Repository:    view.Scope.Repository,
		ViewHash:      view.ViewHash,
		LockHash:      view.LockHash,
		Profile:       view.Profile,
		Levels:        view.Levels,
		Overlays:      view.Overlays,
		Conflicts:     conflicts,
		Frameworks:    frameworks,
		UnmappedRules: unmapped,
		Exceptions:    exceptions,
		Evidence:      evRecords,
	}, nil
}

func loadLocalEvidence(rootDir string) []evidence.Record {
	if strings.TrimSpace(rootDir) == "" {
		return nil
	}
	evDir := filepath.Join(rootDir, ".gradus", "evidence")
	entries, err := os.ReadDir(evDir)
	if err != nil {
		return nil
	}
	var out []evidence.Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(evDir, e.Name()))
		if err != nil {
			continue
		}
		var rec evidence.Record
		if err := json.Unmarshal(raw, &rec); err == nil && rec.ID != "" {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Timestamp != out[j].Timestamp {
			return out[i].Timestamp < out[j].Timestamp
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// RenderMarkdown renders a deterministic Markdown audit report from an AuditBundle.
func RenderMarkdown(b AuditBundle) string {
	var sb strings.Builder
	sb.WriteString("# Gradus Compliance & Audit Export\n\n")
	sb.WriteString(fmt.Sprintf("- **Schema:** `%s`\n", b.Schema))
	sb.WriteString(fmt.Sprintf("- **Generated At:** `%s`\n", b.GeneratedAt))
	sb.WriteString(fmt.Sprintf("- **Repository:** `%s`\n", b.Repository))
	sb.WriteString(fmt.Sprintf("- **View Hash:** `%s`\n", b.ViewHash))
	if b.LockHash != "" {
		sb.WriteString(fmt.Sprintf("- **Lock Hash:** `%s`\n", b.LockHash))
	}
	sb.WriteString(fmt.Sprintf("- **Blocking Conflicts:** `%d`\n\n", len(b.Conflicts)))

	sb.WriteString("## 1. Framework Control Coverage\n\n")
	if len(b.Frameworks) == 0 {
		sb.WriteString("_No framework controls mapped._\n\n")
	} else {
		for _, fw := range b.Frameworks {
			sb.WriteString(fmt.Sprintf("### %s (%d controls, %d rules)\n\n", fw.Framework, fw.ControlCount, fw.RuleCount))
			sb.WriteString("| Control | Rule ID | Level | Severity | Enforcement | Waived | Source |\n")
			sb.WriteString("|---|---|---|---|---|---|---|\n")
			for _, c := range fw.Controls {
				for _, r := range c.Rules {
					sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%t` | `%s` |\n",
						c.Control, r.RuleID, r.Level, r.Severity, r.Enforcement, r.Waived, r.Source))
				}
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## 2. Unmapped Active Rules\n\n")
	if len(b.UnmappedRules) == 0 {
		sb.WriteString("_All active rules are mapped to at least one control._\n\n")
	} else {
		for _, id := range b.UnmappedRules {
			sb.WriteString(fmt.Sprintf("- `%s`\n", id))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 3. Active Exceptions\n\n")
	if len(b.Exceptions) == 0 {
		sb.WriteString("_No active exceptions._\n\n")
	} else {
		for _, ex := range b.Exceptions {
			sb.WriteString(fmt.Sprintf("- `%s` (`%s`, rule `%s`, expires `%s`, approved by: `%s`): %s\n",
				ex.ID, ex.Kind, ex.RuleID, ex.Expires, strings.Join(ex.ApprovedBy, ", "), ex.Reason))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 4. Recorded Evidence\n\n")
	if len(b.Evidence) == 0 {
		sb.WriteString("_No local evidence records found._\n")
	} else {
		sb.WriteString("| Evidence ID | Kind | Verdict | Timestamp | Commit |\n")
		sb.WriteString("|---|---|---|---|---|\n")
		for _, ev := range b.Evidence {
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` |\n",
				ev.ID, ev.Kind, ev.Verdict, ev.Timestamp, ev.Commit))
		}
	}
	return sb.String()
}
