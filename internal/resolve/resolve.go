package resolve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/source"
)

// Options configures pyramid resolution.
type Options struct {
	IncludeDraft bool
	InCI         bool
	OfflineOnly  bool
	Now          time.Time
}

// ResolveResult returns both the ResolvedView and the computed Lockfile ready for --write.
type ResolveResult struct {
	View     ResolvedView
	Lockfile lock.Lockfile
}

type exceptionRecord struct {
	Schema     string   `yaml:"schema"`
	ID         string   `yaml:"id"`
	RuleID     string   `yaml:"rule_id"`
	Scope      []string `yaml:"scope"`
	Reason     string   `yaml:"reason"`
	ApprovedBy []string `yaml:"approved_by"`
	Created    string   `yaml:"created"`
	Expires    string   `yaml:"expires"`
	Kind       string   `yaml:"kind"`
}

type findingRecord struct {
	Schema  string `yaml:"schema"`
	ID      string `yaml:"id"`
	RuleA   string `yaml:"rule_a"`
	RuleB   string `yaml:"rule_b"`
	Status  string `yaml:"status"` // suspected | confirmed | dismissed
	Message string `yaml:"message"`
}

// Resolve executes the §10 resolution algorithm for rootDir.
func Resolve(rootDir string, opts Options) (ResolveResult, error) {
	var out ResolveResult
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	inCI := opts.InCI || os.Getenv("CI") != ""

	cfgPath := filepath.Join(rootDir, "gradus.yaml")
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return out, fmt.Errorf("failed to read %s: %w", cfgPath, err)
	}
	defHash := lock.HashBytes(cfgBytes)

	cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
	if err != nil {
		return out, err
	}
	cfg := cfgRes.Value

	var notes []string
	notes = append(notes, cfgRes.Notes...)

	// Read existing gradus.lock if present.
	lockPath := filepath.Join(rootDir, "gradus.lock")
	var existingLock *lock.Lockfile
	if lf, err := lock.ReadLockfile(lockPath); err == nil {
		existingLock = &lf
	} else if !os.IsNotExist(err) {
		return out, err
	}

	var lockEntries []lock.Entry
	var levelSummaries []LevelSummary
	var overlaySummaries []OverlaySummary
	var levelSteps []PyramidStep
	var overlaySteps []PyramidStep
	levelConfigs := make(map[string]*schema.LevelConfig)
	levelMeta := make(map[string]LevelMeta)
	includesUser := false

	// Resolve levels.
	for _, lvl := range cfg.Levels {
		var lockEntry *lock.Entry
		if existingLock != nil {
			if e, ok := existingLock.FindEntry("level", lvl.ID); ok {
				lockEntry = &e
			}
		}

		resSrc, err := resolveSource(rootDir, "level", lvl.ID, lvl.Source, lvl.Offline, lockEntry, inCI, opts.OfflineOnly)
		if err != nil {
			return out, err
		}
		notes = append(notes, resSrc.Notes...)
		if resSrc.Skipped {
			continue
		}

		if lvl.Locked && lockEntry != nil {
			if err := lock.VerifyLockedEntry(*lockEntry, resSrc.Hash); err != nil {
				return out, err
			}
		}

		if lvl.ID == "user" {
			includesUser = true
		}

		var lc *schema.LevelConfig
		lvlYAMLPath := filepath.Join(resSrc.Dir, "level.yaml")
		if lvlBytes, err := os.ReadFile(lvlYAMLPath); err == nil {
			parsed, parseErr := schema.ParseLevelConfig(lvlBytes)
			if parseErr != nil {
				return out, fmt.Errorf("level %q (%s): %w", lvl.ID, lvlYAMLPath, parseErr)
			}
			notes = append(notes, parsed.Notes...)
			lc = &parsed.Value
			levelConfigs[lvl.ID] = lc
		} else if !os.IsNotExist(err) {
			return out, fmt.Errorf("failed to read %s: %w", lvlYAMLPath, err)
		}

		ver := ""
		if lc != nil {
			ver = lc.Version
		}
		levelSummaries = append(levelSummaries, LevelSummary{
			ID:      lvl.ID,
			Version: ver,
			Hash:    resSrc.Hash,
		})

		if lvl.ID != "user" {
			sig := ""
			if resSrc.Commit != "" {
				sig = "none"
			}
			lockEntries = append(lockEntries, lock.Entry{
				Kind:      "level",
				ID:        lvl.ID,
				Source:    resSrc.SourceSpec,
				Ref:       resSrc.Ref,
				Commit:    resSrc.Commit,
				Hash:      resSrc.Hash,
				Signature: sig,
			})
		}

		levelMeta[lvl.ID] = LevelMeta{
			ID:            lvl.ID,
			Locked:        lvl.Locked,
			EditableRules: lvl.EditableRules,
		}

		provSource := lvl.ID
		if ver != "" {
			provSource = fmt.Sprintf("%s@v%s", lvl.ID, strings.TrimPrefix(ver, "v"))
		}
		levelSteps = append(levelSteps, PyramidStep{
			Kind:          "level",
			ID:            lvl.ID,
			Locked:        lvl.Locked,
			EditableRules: lvl.EditableRules,
			Dir:           resSrc.Dir,
			SourceLabel:   provSource,
			LevelConfig:   lc,
		})
	}

	// Resolve overlays.
	for _, ov := range cfg.Overlays {
		var lockEntry *lock.Entry
		if existingLock != nil {
			if e, ok := existingLock.FindEntry("overlay", ov.ID); ok {
				lockEntry = &e
			}
		}

		resSrc, err := resolveSource(rootDir, "overlay", ov.ID, ov.Source, nil, lockEntry, inCI, opts.OfflineOnly)
		if err != nil {
			return out, err
		}
		notes = append(notes, resSrc.Notes...)
		if resSrc.Skipped {
			continue
		}

		ovYAMLPath := filepath.Join(resSrc.Dir, "overlay.yaml")
		if ovBytes, err := os.ReadFile(ovYAMLPath); err == nil {
			parsed, parseErr := schema.ParseOverlayConfig(ovBytes)
			if parseErr != nil {
				return out, fmt.Errorf("overlay %q (%s): %w", ov.ID, ovYAMLPath, parseErr)
			}
			notes = append(notes, parsed.Notes...)
		}

		active := ov.Activation == "" || ov.Activation == "always"
		overlaySummaries = append(overlaySummaries, OverlaySummary{
			ID:       ov.ID,
			Strength: ov.Strength,
			Active:   active,
		})

		lockEntries = append(lockEntries, lock.Entry{
			Kind:   "overlay",
			ID:     ov.ID,
			Source: resSrc.SourceSpec,
			Ref:    resSrc.Ref,
			Commit: resSrc.Commit,
			Hash:   resSrc.Hash,
		})

		if active {
			overlaySteps = append(overlaySteps, PyramidStep{
				Kind:        "overlay",
				ID:          ov.ID,
				Strength:    ov.Strength,
				Dir:         resSrc.Dir,
				SourceLabel: fmt.Sprintf("overlay:%s", ov.ID),
			})
		}
	}

	var conflicts []Conflict
	conflicts = append(conflicts, CheckInheritedConstraints(cfg, levelConfigs)...)

	orderedSteps, ordConflicts := OrderPyramidSteps(levelSteps, overlaySteps)
	conflicts = append(conflicts, ordConflicts...)

	// Load rules in ordered sequence.
	var rawRules []ResolvedRule
	for _, step := range orderedSteps {
		rulesDir := filepath.Join(step.Dir, "rules")
		entries, err := os.ReadDir(rulesDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return out, fmt.Errorf("failed to read rules in %s: %w", rulesDir, err)
		}

		var mdFiles []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
				mdFiles = append(mdFiles, e.Name())
			}
		}
		sort.Strings(mdFiles)

		derivedLevel := step.ID
		if step.Kind == "overlay" {
			derivedLevel = step.Strength
		}

		for _, name := range mdFiles {
			rulePath := filepath.Join(rulesDir, name)
			if linfo, lerr := os.Lstat(rulePath); lerr == nil && linfo.Mode()&os.ModeSymlink != 0 {
				return out, fmt.Errorf("symlink rule file %s is not allowed (Threat Model T2)", rulePath)
			}
			rawBytes, err := os.ReadFile(rulePath)
			if err != nil {
				return out, err
			}
			parsed, err := schema.ParseRuleBytes(rawBytes, rulePath, derivedLevel, step.SourceLabel)
			if err != nil {
				return out, err
			}
			notes = append(notes, parsed.Notes...)

			r := parsed.Value
			if r.Status == "retired" {
				continue
			}
			if r.Status == "draft" && !opts.IncludeDraft {
				continue
			}

			var ovPtr *string
			if r.Overrides != "" {
				s := r.Overrides
				ovPtr = &s
			}

			rawRules = append(rawRules, ResolvedRule{
				ID:               r.ID,
				Title:            r.Title,
				Level:            r.Level,
				Severity:         r.Severity,
				Enforcement:      r.Enforcement,
				Scope:            r.Scope,
				Text:             r.Statement,
				Why:              r.Why,
				Rationalizations: r.Rationalizations,
				Overrides:        ovPtr,
				Overridable:      r.Overridable,
				Checker:          r.Checker,
				Guard:            r.Guard,
				IsOverlay:        step.Kind == "overlay",
				Provenance: RuleProvenance{
					Source: step.SourceLabel,
					Path:   filepath.ToSlash(filepath.Join("rules", name)),
					Hash:   lock.HashBytes(rawBytes),
				},
			})
		}
	}

	mergedRules, mergeConflicts, err := MergeAndCheckRules(rawRules, levelMeta)
	if err != nil {
		return out, err
	}
	conflicts = append(conflicts, mergeConflicts...)

	// Load exceptions (.gradus/exceptions/*.yaml).
	activeExceptions, expNotes, err := loadExceptions(filepath.Join(rootDir, ".gradus", "exceptions"), now)
	if err != nil {
		return out, err
	}
	notes = append(notes, expNotes...)

	for i := range mergedRules {
		for _, ex := range activeExceptions {
			if ex.RuleID == mergedRules[i].ID {
				mergedRules[i].Waived = true
				mergedRules[i].WaiverIDs = append(mergedRules[i].WaiverIDs, ex.ID)
			}
		}
	}

	// Load confirmed semantic conflicts (.gradus/findings/*.yaml).
	confirmedFindings, err := loadConfirmedFindings(filepath.Join(rootDir, ".gradus", "findings"))
	if err != nil {
		return out, err
	}
	conflicts = append(conflicts, confirmedFindings...)

	if conflicts == nil {
		conflicts = []Conflict{}
	}
	if activeExceptions == nil {
		activeExceptions = []ExceptionView{}
	}
	if mergedRules == nil {
		mergedRules = []ResolvedRule{}
	}

	charCount := 0
	for _, r := range mergedRules {
		charCount += len(r.ID) + len(r.Title) + len(r.Text) + len(r.Why)
		for _, rat := range r.Rationalizations {
			charCount += len(rat)
		}
	}

	lockHash := ""
	if existingLock != nil {
		if b, err := os.ReadFile(lockPath); err == nil {
			lockHash = lock.HashBytes(b)
		}
	}

	view := ResolvedView{
		Schema:   schema.SchemaVersion,
		LockHash: lockHash,
		Scope: ViewScope{
			Repository:        ".",
			Path:              nil,
			IncludesUserLevel: includesUser,
		},
		Levels:     levelSummaries,
		Overlays:   overlaySummaries,
		Profile:    cfg.Profile,
		Rules:      mergedRules,
		Conflicts:  conflicts,
		Exceptions: activeExceptions,
		Budget: BudgetInfo{
			Characters: charCount,
			Targets: map[string]TargetLimit{
				"antigravity": {Limit: 12000},
			},
		},
		Notes: notes,
	}

	viewBytes, _ := json.Marshal(view)
	view.ViewHash = lock.HashBytes(viewBytes)

	out.View = view
	out.Lockfile = lock.Lockfile{
		Schema:         schema.SchemaVersion,
		DefinitionHash: defHash,
		Entries:        lockEntries,
		ViewHash:       view.ViewHash,
	}
	return out, nil
}

func resolveSource(rootDir, kind, id string, src schema.SourceRef, offline *schema.OfflinePolicy, lockEntry *lock.Entry, inCI, offlineOnly bool) (source.ResolvedSource, error) {
	if src.Git != "" {
		if offlineOnly {
			return source.ResolveGitSourceOffline(rootDir, kind, id, src, offline, lockEntry)
		}
		return source.ResolveGitSource(rootDir, kind, id, src, offline, lockEntry)
	}
	return source.ResolveLocalSource(rootDir, kind, id, src.Path, inCI)
}

func loadExceptions(dir string, now time.Time) ([]ExceptionView, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var active []ExceptionView
	var notes []string
	today := now.Format("2006-01-02")

	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, nil, err
		}
		var ex exceptionRecord
		if err := yaml.Unmarshal(data, &ex); err != nil {
			return nil, nil, fmt.Errorf("invalid exception %s: %w", e.Name(), err)
		}
		if ex.Expires != "" && ex.Expires < today {
			notes = append(notes, fmt.Sprintf("exception %s for rule %s expired on %s", ex.ID, ex.RuleID, ex.Expires))
			continue
		}
		active = append(active, ExceptionView{
			ID:         ex.ID,
			RuleID:     ex.RuleID,
			Scope:      ex.Scope,
			Kind:       ex.Kind,
			Expires:    ex.Expires,
			ApprovedBy: ex.ApprovedBy,
		})
	}
	return active, notes, nil
}

func loadConfirmedFindings(dir string) ([]Conflict, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var conflicts []Conflict
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var f findingRecord
		if err := yaml.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("invalid finding %s: %w", e.Name(), err)
		}
		if f.Status == "confirmed" {
			conflicts = append(conflicts, Conflict{
				Kind:    "semantic",
				RuleID:  f.RuleA,
				Target:  f.RuleB,
				Message: fmt.Sprintf("confirmed semantic conflict (%s): %s", f.ID, f.Message),
			})
		}
	}
	return conflicts, nil
}
