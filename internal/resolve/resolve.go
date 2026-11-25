package resolve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/source"
)

// Options configures pyramid resolution.
type Options struct {
	IncludeDraft    bool
	InCI            bool
	OfflineOnly     bool
	AllowLockUpdate bool
	AgentProfile    string
	Now             time.Time
}

// ResolveResult returns both the ResolvedView and the computed Lockfile ready for --write.
type ResolveResult struct {
	View     ResolvedView
	Lockfile lock.Lockfile
}

type exceptionRecord struct {
	Schema        string   `yaml:"schema"`
	ID            string   `yaml:"id"`
	RuleID        string   `yaml:"rule_id"`
	Scope         []string `yaml:"scope"`
	Reason        string   `yaml:"reason"`
	ApprovedBy    []string `yaml:"approved_by"`
	Created       string   `yaml:"created"`
	Expires       string   `yaml:"expires"`
	Kind          string   `yaml:"kind"`
	PostReviewDue string   `yaml:"post_review_due,omitempty"`
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
	agentProfile := strings.TrimSpace(opts.AgentProfile)
	if agentProfile == "" {
		agentProfile = strings.TrimSpace(os.Getenv("GRADUS_AGENT_PROFILE"))
	}
	if agentProfile != "" {
		info, serr := os.Stat(agentProfile)
		if serr != nil || !info.IsDir() {
			return out, fmt.Errorf("agent-profile directory %q not found or not a directory", agentProfile)
		}
		var filteredLevels []schema.LevelRef
		for _, l := range cfg.Levels {
			if l.ID == "user" || strings.HasPrefix(l.Source.Path, "~/") {
				continue
			}
			filteredLevels = append(filteredLevels, l)
		}
		filteredLevels = append(filteredLevels, schema.LevelRef{
			ID:     "agent-profile",
			Source: schema.SourceRef{Path: agentProfile},
		})
		cfg.Levels = filteredLevels
	}

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
			refChanged := opts.AllowLockUpdate && lvl.Source.Git != "" && lockEntry.Ref != "" && lockEntry.Ref != resSrc.Ref
			if !refChanged {
				if err := lock.VerifyLockedEntry(*lockEntry, resSrc.Hash); err != nil {
					return out, err
				}
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
		var charter *schema.Charter
		lcSig := ""
		lcMinVer := ""
		if lc != nil {
			ver = lc.Version
			charter = lc.Charter
			lcSig = lc.Signature
			lcMinVer = lc.MinVersion
		}

		if err := enforceMinVersion(lvl.ID, lvl.MinVersion, lcMinVer, ver, resSrc.Ref); err != nil {
			return out, err
		}

		sig := ""
		if resSrc.Commit != "" {
			sigPolicy := effectiveSignaturePolicy(lvl.Signature, lcSig)
			if opts.OfflineOnly && lockEntry != nil && lockEntry.Signature == "verified" && lockEntry.Commit == resSrc.Commit {
				sig = "verified"
			} else {
				verifiedSig, sigErr := source.VerifyGitSignature(resSrc.Dir, resSrc.Ref)
				if sigErr != nil {
					if sigPolicy == "required" {
						return out, fmt.Errorf("level %q: signature verification failed: %w", lvl.ID, sigErr)
					}
					sig = "none"
				} else {
					sig = verifiedSig
				}
			}
		}

		levelSummaries = append(levelSummaries, LevelSummary{
			ID:      lvl.ID,
			Version: ver,
			Hash:    resSrc.Hash,
			Charter: charter,
		})

		if lvl.ID != "user" {
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

	// Collect level-declared activation_rules in pyramid order.
	var activationRules []LevelActivationRule
	for _, step := range levelSteps {
		if step.LevelConfig != nil {
			for _, ar := range step.LevelConfig.ActivationRules {
				activationRules = append(activationRules, LevelActivationRule{
					LevelID: step.ID,
					Rule:    ar,
				})
			}
		}
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

		var ovCfg *schema.OverlayConfig
		ovYAMLPath := filepath.Join(resSrc.Dir, "overlay.yaml")
		if ovBytes, err := os.ReadFile(ovYAMLPath); err == nil {
			parsed, parseErr := schema.ParseOverlayConfig(ovBytes)
			if parseErr != nil {
				return out, fmt.Errorf("overlay %q (%s): %w", ov.ID, ovYAMLPath, parseErr)
			}
			notes = append(notes, parsed.Notes...)
			ovCfg = &parsed.Value
		}

		active, reason := EvaluateOverlayActivation(ov, ovCfg, cfg.Profile, activationRules)
		overlaySummaries = append(overlaySummaries, OverlaySummary{
			ID:       ov.ID,
			Strength: ov.Strength,
			Active:   active,
			Reason:   reason,
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
	conflicts = append(conflicts, CheckMandatoryOverlays(cfg.Profile, activationRules, overlaySummaries)...)

	orderedSteps, ordConflicts := OrderPyramidSteps(levelSteps, overlaySteps)
	conflicts = append(conflicts, ordConflicts...)

	// Load rules in ordered sequence.
	var rawRules []ResolvedRule
	var inactiveRules []ResolvedRule
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
			var ovPtr *string
			if r.Overrides != "" {
				s := r.Overrides
				ovPtr = &s
			}

			resolved := ResolvedRule{
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
				Status:           r.Status,
				ReviewBy:         r.ReviewBy,
				Expires:          r.Expires,
				Controls:         r.Controls,
				IsOverlay:        step.Kind == "overlay",
				Provenance: RuleProvenance{
					Source: step.SourceLabel,
					Path:   filepath.ToSlash(filepath.Join("rules", name)),
					Hash:   lock.HashBytes(rawBytes),
				},
			}
			if r.Status == "retired" || (r.Status == "draft" && !opts.IncludeDraft) {
				inactiveRules = append(inactiveRules, resolved)
				continue
			}
			rawRules = append(rawRules, resolved)
		}
	}

	mergedRules, mergeConflicts, err := MergeAndCheckRules(rawRules, levelMeta)
	if err != nil {
		return out, err
	}
	conflicts = append(conflicts, mergeConflicts...)
	conflicts = append(conflicts, checkCharterAndBudget(mergedRules, levelConfigs, levelSummaries)...)
	lifecycleWarnings, lifecycleConflicts := checkRuleLifecycle(mergedRules, now)
	conflicts = append(conflicts, lifecycleConflicts...)

	// Load exceptions (.gradus/exceptions/*.yaml).
	activeExceptions, expNotes, err := loadExceptions(filepath.Join(rootDir, ".gradus", "exceptions"), now)
	if err != nil {
		return out, err
	}
	notes = append(notes, expNotes...)
	conflicts = append(conflicts, validateExceptionPolicies(cfg, levelConfigs, mergedRules, activeExceptions, now)...)

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

	resolvedObligations, resolvedBindings, obConflicts := resolveObligationsAndBindings(cfg, levelSteps)
	conflicts = append(conflicts, obConflicts...)

	resolvedToolchain, tcConflicts := resolveToolchain(cfg, levelSteps)
	conflicts = append(conflicts, tcConflicts...)

	charCount := 0
	for _, r := range mergedRules {
		charCount += ruleCharCount(r)
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
		Levels:        levelSummaries,
		Overlays:      overlaySummaries,
		Profile:       cfg.Profile,
		Rules:         mergedRules,
		InactiveRules: inactiveRules,
		Conflicts:     conflicts,
		Exceptions:    activeExceptions,
		Budget: BudgetInfo{
			Characters: charCount,
			Targets: map[string]TargetLimit{
				"antigravity": {Limit: 12000},
			},
		},
		Obligations: resolvedObligations,
		Bindings:    resolvedBindings,
		Toolchain:   resolvedToolchain,
		Warnings:    lifecycleWarnings,
		Notes:       notes,
	}

	view.ViewHash = computeProjectViewHash(view)

	out.View = view
	out.Lockfile = lock.Lockfile{
		Schema:         schema.SchemaVersion,
		DefinitionHash: defHash,
		Entries:        lockEntries,
		ViewHash:       view.ViewHash,
	}
	return out, nil
}

// computeProjectViewHash computes a deterministic project-only hash excluding user-level
// rules and level summaries so gradus.lock and compiled project markers never depend on ~/.gradus/levels/user (R2.AC1).
func computeProjectViewHash(view ResolvedView) string {
	proj := view
	proj.ViewHash = ""
	proj.LockHash = ""
	proj.Scope.IncludesUserLevel = false

	var projLevels []LevelSummary
	for _, l := range view.Levels {
		if l.ID != "user" {
			projLevels = append(projLevels, l)
		}
	}
	proj.Levels = projLevels

	filterRules := func(in []ResolvedRule) []ResolvedRule {
		var out []ResolvedRule
		for _, r := range in {
			if r.Level != "user" {
				out = append(out, r)
			}
		}
		return out
	}
	proj.Rules = filterRules(view.Rules)
	proj.InactiveRules = filterRules(view.InactiveRules)

	projChars := 0
	for _, r := range proj.Rules {
		projChars += ruleCharCount(r)
	}
	proj.Budget.Characters = projChars

	b, _ := json.Marshal(proj)
	return lock.HashBytes(b)
}

func ruleCharCount(r ResolvedRule) int {
	n := len(r.ID) + len(r.Title) + len(r.Text) + len(r.Why)
	for _, rat := range r.Rationalizations {
		n += len(rat)
	}
	return n
}

func checkCharterAndBudget(rules []ResolvedRule, levelConfigs map[string]*schema.LevelConfig, levelSummaries []LevelSummary) []Conflict {
	var conflicts []Conflict
	ruleCounts := make(map[string]int)
	charCounts := make(map[string]int)

	for _, r := range rules {
		if r.IsOverlay {
			continue
		}
		ruleCounts[r.Level]++
		charCounts[r.Level] += ruleCharCount(r)

		lc := levelConfigs[r.Level]
		if lc != nil && lc.Charter != nil && len(lc.Charter.AllowedSeverities) > 0 {
			allowed := false
			for _, sev := range lc.Charter.AllowedSeverities {
				if r.Severity == sev {
					allowed = true
					break
				}
			}
			if !allowed {
				conflicts = append(conflicts, Conflict{
					Kind:    "charter",
					RuleID:  r.ID,
					Level:   r.Level,
					Message: fmt.Sprintf("rule %s uses severity %s, which is not allowed by level %q charter (allowed: %s)", r.ID, r.Severity, r.Level, strings.Join(lc.Charter.AllowedSeverities, ", ")),
				})
			}
		}
	}

	for _, ls := range levelSummaries {
		lc := levelConfigs[ls.ID]
		if lc == nil || lc.ContextBudget == nil {
			continue
		}
		if lc.ContextBudget.MaxRules != nil && ruleCounts[ls.ID] > *lc.ContextBudget.MaxRules {
			conflicts = append(conflicts, Conflict{
				Kind:    "budget",
				Level:   ls.ID,
				Message: fmt.Sprintf("level %q exceeds context_budget.max_rules (%d > %d)", ls.ID, ruleCounts[ls.ID], *lc.ContextBudget.MaxRules),
			})
		}
		if lc.ContextBudget.MaxChars != nil && charCounts[ls.ID] > *lc.ContextBudget.MaxChars {
			conflicts = append(conflicts, Conflict{
				Kind:    "budget",
				Level:   ls.ID,
				Message: fmt.Sprintf("level %q exceeds context_budget.max_chars (%d > %d)", ls.ID, charCounts[ls.ID], *lc.ContextBudget.MaxChars),
			})
		}
	}
	return conflicts
}

func checkRuleLifecycle(rules []ResolvedRule, now time.Time) ([]string, []Conflict) {
	today := now.UTC().Format("2006-01-02")
	var warnings []string
	var conflicts []Conflict

	for _, r := range rules {
		if r.Status == "deprecated" {
			warnings = append(warnings, fmt.Sprintf("rule %s (%s) is deprecated", r.ID, r.Level))
		}
		if r.ReviewBy != "" && r.ReviewBy < today {
			warnings = append(warnings, fmt.Sprintf("rule %s (%s) review_by date %s has passed", r.ID, r.Level, r.ReviewBy))
		}
		if r.Expires != "" && r.Expires < today {
			conflicts = append(conflicts, Conflict{
				Kind:    "lifecycle",
				RuleID:  r.ID,
				Level:   r.Level,
				Message: fmt.Sprintf("rule %s (%s) expired on %s", r.ID, r.Level, r.Expires),
			})
		}
	}
	return warnings, conflicts
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
		if (ex.Kind == "" || ex.Kind == "exception") && len(ex.ApprovedBy) == 0 {
			notes = append(notes, fmt.Sprintf("exception %s for rule %s is pending approval", ex.ID, ex.RuleID))
			continue
		}
		active = append(active, ExceptionView{
			ID:            ex.ID,
			RuleID:        ex.RuleID,
			Scope:         ex.Scope,
			Kind:          ex.Kind,
			Reason:        ex.Reason,
			Created:       ex.Created,
			Expires:       ex.Expires,
			PostReviewDue: ex.PostReviewDue,
			ApprovedBy:    ex.ApprovedBy,
		})
	}
	return active, notes, nil
}

// ParseDurationSpan parses durations like "30d" or "72h".
func ParseDurationSpan(raw string) (time.Duration, error) {
	s := strings.TrimSpace(raw)
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil || days < 0 {
			return 0, fmt.Errorf("invalid day duration %q", raw)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// ValidateExceptionAgainstLevel validates a single exception against its target level policy.
func ValidateExceptionAgainstLevel(ex ExceptionView, lvlID string, lc *schema.LevelConfig, offline *schema.OfflinePolicy, now time.Time) []Conflict {
	var conflicts []Conflict
	today := now.UTC().Format("2006-01-02")

	if lc != nil && lc.Exceptions != nil && lc.Exceptions.MaxDuration != "" && ex.Created != "" && ex.Expires != "" {
		maxDur, err := ParseDurationSpan(lc.Exceptions.MaxDuration)
		if err == nil {
			createdAt, errC := time.Parse("2006-01-02", ex.Created)
			expiresAt, errE := time.Parse("2006-01-02", ex.Expires)
			if errC == nil && errE == nil && expiresAt.Sub(createdAt) > maxDur {
				conflicts = append(conflicts, Conflict{
					Kind:    "exception",
					RuleID:  ex.RuleID,
					Level:   lvlID,
					Message: fmt.Sprintf("exception %s for rule %s exceeds level %q exceptions.max_duration (%s)", ex.ID, ex.RuleID, lvlID, lc.Exceptions.MaxDuration),
				})
			}
		}
	}

	if ex.Kind == "break-glass" {
		if strings.TrimSpace(ex.Reason) == "" {
			conflicts = append(conflicts, Conflict{
				Kind:    "exception",
				RuleID:  ex.RuleID,
				Level:   lvlID,
				Message: fmt.Sprintf("break-glass %s for rule %s requires a non-empty reason", ex.ID, ex.RuleID),
			})
		}
		if strings.TrimSpace(ex.PostReviewDue) == "" {
			conflicts = append(conflicts, Conflict{
				Kind:    "exception",
				RuleID:  ex.RuleID,
				Level:   lvlID,
				Message: fmt.Sprintf("break-glass %s for rule %s requires post_review_due", ex.ID, ex.RuleID),
			})
		} else if _, err := time.Parse("2006-01-02", ex.PostReviewDue); err != nil {
			conflicts = append(conflicts, Conflict{
				Kind:    "exception",
				RuleID:  ex.RuleID,
				Level:   lvlID,
				Message: fmt.Sprintf("break-glass %s post_review_due %q must be YYYY-MM-DD", ex.ID, ex.PostReviewDue),
			})
		} else if ex.PostReviewDue < today {
			conflicts = append(conflicts, Conflict{
				Kind:    "exception",
				RuleID:  ex.RuleID,
				Level:   lvlID,
				Message: fmt.Sprintf("break-glass %s for rule %s post_review_due %s has passed", ex.ID, ex.RuleID, ex.PostReviewDue),
			})
		}
		if offline != nil && offline.Mode == "fail-closed" {
			conflicts = append(conflicts, Conflict{
				Kind:    "exception",
				RuleID:  ex.RuleID,
				Level:   lvlID,
				Message: fmt.Sprintf("break-glass %s is disallowed on fail-closed level %q", ex.ID, lvlID),
			})
		} else if lc != nil && lc.Exceptions != nil && !lc.Exceptions.BreakGlass {
			conflicts = append(conflicts, Conflict{
				Kind:    "exception",
				RuleID:  ex.RuleID,
				Level:   lvlID,
				Message: fmt.Sprintf("break-glass %s is disallowed by level %q exceptions.break_glass: false", ex.ID, lvlID),
			})
		}
	}

	return conflicts
}

func validateExceptionPolicies(cfg schema.PyramidConfig, levelConfigs map[string]*schema.LevelConfig, rules []ResolvedRule, exceptions []ExceptionView, now time.Time) []Conflict {
	ruleLevel := make(map[string]string, len(rules))
	for _, r := range rules {
		ruleLevel[r.ID] = r.Level
	}
	offlineByLevel := make(map[string]*schema.OfflinePolicy, len(cfg.Levels))
	for _, l := range cfg.Levels {
		offlineByLevel[l.ID] = l.Offline
	}
	var conflicts []Conflict
	for _, ex := range exceptions {
		lvl := ruleLevel[ex.RuleID]
		if lvl == "" {
			continue
		}
		conflicts = append(conflicts, ValidateExceptionAgainstLevel(ex, lvl, levelConfigs[lvl], offlineByLevel[lvl], now)...)
	}
	return conflicts
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

func effectiveSignaturePolicy(cfgPolicy, levelPolicy string) string {
	if cfgPolicy == "required" || levelPolicy == "required" {
		return "required"
	}
	if cfgPolicy == "optional" || levelPolicy == "optional" {
		return "optional"
	}
	return "optional"
}

func enforceMinVersion(levelID, cfgMinVer, levelMinVer, resolvedVer, ref string) error {
	effectiveMin := cfgMinVer
	if levelMinVer != "" {
		if effectiveMin == "" {
			effectiveMin = levelMinVer
		} else {
			a, errA := schema.ParseSemVer(effectiveMin)
			b, errB := schema.ParseSemVer(levelMinVer)
			if errA == nil && errB == nil && schema.CompareSemVer(b, a) > 0 {
				effectiveMin = levelMinVer
			}
		}
	}
	if effectiveMin == "" {
		return nil
	}

	minSV, err := schema.ParseSemVer(effectiveMin)
	if err != nil {
		return fmt.Errorf("level %q: invalid min_version %q: %w", levelID, effectiveMin, err)
	}

	candidate := strings.TrimSpace(resolvedVer)
	if candidate == "" {
		candidate = strings.TrimSpace(ref)
	}
	resolvedSV, err := schema.ParseSemVer(candidate)
	if err != nil {
		return fmt.Errorf("level %q: resolved version %q is not valid SemVer (required min_version %s): %w", levelID, candidate, effectiveMin, err)
	}
	if schema.CompareSemVer(resolvedSV, minSV) < 0 {
		return fmt.Errorf("level %q: resolved version %s is below required min_version %s", levelID, candidate, effectiveMin)
	}
	return nil
}
