package changelog

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

// MetadataChange records a level metadata difference between two refs.
type MetadataChange struct {
	Field string `json:"field"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
}

// RuleSummary describes an added or removed rule in a level changelog.
type RuleSummary struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Severity    string   `json:"severity"`
	Enforcement string   `json:"enforcement"`
	Status      string   `json:"status,omitempty"`
	Scope       []string `json:"scope,omitempty"`
}

// FieldChange describes a single changed attribute on a modified rule.
type FieldChange struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// RuleModification describes a rule present in both refs with changed fields.
type RuleModification struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Changes []FieldChange `json:"changes"`
}

// Report represents the deterministic semantic diff between two versions of a level.
type Report struct {
	LevelID         string             `json:"level_id"`
	FromRef         string             `json:"from_ref"`
	ToRef           string             `json:"to_ref"`
	MetadataChanges []MetadataChange   `json:"metadata_changes"`
	Added           []RuleSummary      `json:"added"`
	Removed         []RuleSummary      `json:"removed"`
	Modified        []RuleModification `json:"modified"`
}

type levelSnapshot struct {
	Config *schema.LevelConfig
	Rules  map[string]schema.Rule
}

// DiffLevel materializes fromRef and toRef from repoDir into temporary directories
// using `git archive` (non-destructive, PRJ-GO-001) and computes a deterministic diff.
func DiffLevel(repoDir, levelID, fromRef, toRef string) (Report, error) {
	rep := Report{
		LevelID:         levelID,
		FromRef:         strings.TrimSpace(fromRef),
		ToRef:           strings.TrimSpace(toRef),
		MetadataChanges: []MetadataChange{},
		Added:           []RuleSummary{},
		Removed:         []RuleSummary{},
		Modified:        []RuleModification{},
	}

	if err := validateGitRef(rep.FromRef, "from"); err != nil {
		return rep, err
	}
	if err := validateGitRef(rep.ToRef, "to"); err != nil {
		return rep, err
	}

	fromDir, err := os.MkdirTemp("", "gradus-changelog-from-*")
	if err != nil {
		return rep, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(fromDir)

	toDir, err := os.MkdirTemp("", "gradus-changelog-to-*")
	if err != nil {
		return rep, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(toDir)

	if err := extractRefArchive(repoDir, rep.FromRef, fromDir); err != nil {
		return rep, fmt.Errorf("failed to materialize %q: %w", rep.FromRef, err)
	}
	if err := extractRefArchive(repoDir, rep.ToRef, toDir); err != nil {
		return rep, fmt.Errorf("failed to materialize %q: %w", rep.ToRef, err)
	}

	fromSnap, err := loadSnapshot(fromDir, levelID)
	if err != nil {
		return rep, fmt.Errorf("from %q: %w", rep.FromRef, err)
	}
	toSnap, err := loadSnapshot(toDir, levelID)
	if err != nil {
		return rep, fmt.Errorf("to %q: %w", rep.ToRef, err)
	}

	rep.MetadataChanges = diffMetadata(fromSnap.Config, toSnap.Config)

	allIDs := make(map[string]bool)
	for id := range fromSnap.Rules {
		allIDs[id] = true
	}
	for id := range toSnap.Rules {
		allIDs[id] = true
	}
	sortedIDs := make([]string, 0, len(allIDs))
	for id := range allIDs {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)

	for _, id := range sortedIDs {
		oldRule, inOld := fromSnap.Rules[id]
		newRule, inNew := toSnap.Rules[id]
		switch {
		case !inOld && inNew:
			rep.Added = append(rep.Added, summarizeRule(newRule))
		case inOld && !inNew:
			rep.Removed = append(rep.Removed, summarizeRule(oldRule))
		case inOld && inNew:
			if changes := diffRule(oldRule, newRule); len(changes) > 0 {
				rep.Modified = append(rep.Modified, RuleModification{
					ID:      id,
					Title:   newRule.Title,
					Changes: changes,
				})
			}
		}
	}

	return rep, nil
}

// FormatMarkdown renders a human-readable Markdown changelog suitable for PR descriptions.
func FormatMarkdown(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Level Changelog: `%s` (`%s` -> `%s`)\n\n", r.LevelID, r.FromRef, r.ToRef)

	if len(r.MetadataChanges) == 0 && len(r.Added) == 0 && len(r.Removed) == 0 && len(r.Modified) == 0 {
		b.WriteString("No rule or metadata changes detected.\n")
		return b.String()
	}

	if len(r.MetadataChanges) > 0 {
		b.WriteString("## Level Metadata Changes\n\n")
		for _, m := range r.MetadataChanges {
			fmt.Fprintf(&b, "- **%s:** `%s` -> `%s`\n", m.Field, m.From, m.To)
		}
		b.WriteString("\n")
	}

	if len(r.Added) > 0 {
		b.WriteString("## Added Rules\n\n")
		for _, rule := range r.Added {
			fmt.Fprintf(&b, "- **%s** — %s (`%s` / `%s`)\n", rule.ID, rule.Title, rule.Severity, rule.Enforcement)
		}
		b.WriteString("\n")
	}

	if len(r.Removed) > 0 {
		b.WriteString("## Removed Rules\n\n")
		for _, rule := range r.Removed {
			fmt.Fprintf(&b, "- **%s** — %s (`%s` / `%s`)\n", rule.ID, rule.Title, rule.Severity, rule.Enforcement)
		}
		b.WriteString("\n")
	}

	if len(r.Modified) > 0 {
		b.WriteString("## Modified Rules\n\n")
		for _, mod := range r.Modified {
			fmt.Fprintf(&b, "### %s — %s\n", mod.ID, mod.Title)
			for _, ch := range mod.Changes {
				if ch.Field == "text" {
					b.WriteString("- **text:** updated\n")
				} else {
					fmt.Fprintf(&b, "- **%s:** `%s` -> `%s`\n", ch.Field, ch.From, ch.To)
				}
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func validateGitRef(ref, label string) error {
	if ref == "" {
		return fmt.Errorf("%s ref cannot be empty", label)
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("invalid %s ref %q (cannot start with '-')", label, ref)
	}
	return nil
}

func extractRefArchive(repoDir, ref, destDir string) error {
	cmd := exec.Command("git", "archive", "--format=tar", ref, "--")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}

	tr := tar.NewReader(&stdout)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(hdr.Name)
		if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("unsafe archive entry path %q", hdr.Name)
		}
		target := filepath.Join(destDir, clean)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadSnapshot(dir, levelID string) (levelSnapshot, error) {
	snap := levelSnapshot{
		Rules: make(map[string]schema.Rule),
	}

	lvlPath := filepath.Join(dir, "level.yaml")
	if data, err := os.ReadFile(lvlPath); err == nil {
		res, parseErr := schema.ParseLevelConfig(data)
		if parseErr != nil {
			return snap, parseErr
		}
		snap.Config = &res.Value
	} else if !os.IsNotExist(err) {
		return snap, err
	}

	rulesDir := filepath.Join(dir, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return snap, nil
		}
		return snap, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		res, err := schema.ParseRuleFile(filepath.Join(rulesDir, e.Name()), levelID, dir)
		if err != nil {
			return snap, err
		}
		snap.Rules[res.Value.ID] = res.Value
	}
	return snap, nil
}

func diffMetadata(oldCfg, newCfg *schema.LevelConfig) []MetadataChange {
	var changes []MetadataChange
	getVer := func(c *schema.LevelConfig) string {
		if c == nil {
			return ""
		}
		return c.Version
	}
	getMinVer := func(c *schema.LevelConfig) string {
		if c == nil {
			return ""
		}
		return c.MinVersion
	}
	getCharterPurpose := func(c *schema.LevelConfig) string {
		if c == nil || c.Charter == nil {
			return ""
		}
		return c.Charter.Purpose
	}
	getCharterSev := func(c *schema.LevelConfig) string {
		if c == nil || c.Charter == nil {
			return ""
		}
		return strings.Join(c.Charter.AllowedSeverities, ",")
	}

	addIfDiff := func(field, a, b string) {
		if a != b {
			changes = append(changes, MetadataChange{Field: field, From: a, To: b})
		}
	}
	addIfDiff("version", getVer(oldCfg), getVer(newCfg))
	addIfDiff("min_version", getMinVer(oldCfg), getMinVer(newCfg))
	addIfDiff("charter.purpose", getCharterPurpose(oldCfg), getCharterPurpose(newCfg))
	addIfDiff("charter.allowed_severities", getCharterSev(oldCfg), getCharterSev(newCfg))
	return changes
}

func summarizeRule(r schema.Rule) RuleSummary {
	return RuleSummary{
		ID:          r.ID,
		Title:       r.Title,
		Severity:    r.Severity,
		Enforcement: r.Enforcement,
		Status:      r.Status,
		Scope:       r.Scope,
	}
}

func diffRule(oldR, newR schema.Rule) []FieldChange {
	var changes []FieldChange
	add := func(field, a, b string) {
		if a != b {
			changes = append(changes, FieldChange{Field: field, From: a, To: b})
		}
	}
	add("title", oldR.Title, newR.Title)
	add("severity", oldR.Severity, newR.Severity)
	add("enforcement", oldR.Enforcement, newR.Enforcement)
	if !reflect.DeepEqual(oldR.Scope, newR.Scope) {
		add("scope", strings.Join(oldR.Scope, ","), strings.Join(newR.Scope, ","))
	}
	add("status", oldR.Status, newR.Status)
	add("text", strings.TrimSpace(oldR.Statement), strings.TrimSpace(newR.Statement))
	return changes
}
