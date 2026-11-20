package resolve

import "github.com/Milosauro/gradus-ai/internal/schema"

// ResolvedView matches the gradus/v1 resolved view specification (§9.7).
type ResolvedView struct {
	Schema     string           `json:"schema"`
	ViewHash   string           `json:"view_hash"`
	LockHash   string           `json:"lock_hash"`
	Scope      ViewScope        `json:"scope"`
	Levels     []LevelSummary   `json:"levels"`
	Overlays   []OverlaySummary `json:"overlays,omitempty"`
	Profile    *schema.Profile  `json:"profile,omitempty"`
	Rules      []ResolvedRule   `json:"rules"`
	Conflicts  []Conflict       `json:"conflicts"`
	Exceptions []ExceptionView  `json:"exceptions"`
	Budget     BudgetInfo       `json:"budget"`
	Notes      []string         `json:"notes,omitempty"`
}

// ViewScope describes the scope of a resolved view.
type ViewScope struct {
	Repository        string  `json:"repository"`
	Path              *string `json:"path"`
	IncludesUserLevel bool    `json:"includes_user_level"`
}

// LevelSummary summarizes a resolved level.
type LevelSummary struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
	Hash    string `json:"hash"`
}

// OverlaySummary summarizes an overlay in the pyramid.
type OverlaySummary struct {
	ID       string `json:"id"`
	Strength string `json:"strength"`
	Active   bool   `json:"active"`
}

// RuleProvenance records the source file and hash of a resolved rule (§9.7).
type RuleProvenance struct {
	Source string `json:"source"`
	Path   string `json:"path"`
	Hash   string `json:"hash"`
}

// ResolvedRule represents a rule inside the resolved view (§9.7).
type ResolvedRule struct {
	ID               string                `json:"id"`
	Title            string                `json:"title"`
	Level            string                `json:"level"`
	Severity         string                `json:"severity"`
	Enforcement      string                `json:"enforcement"`
	Scope            []string              `json:"scope"`
	Text             string                `json:"text"`
	Why              string                `json:"why,omitempty"`
	Rationalizations []string              `json:"rationalizations,omitempty"`
	Overrides        *string               `json:"overrides"`
	Overridable      *schema.OverridableSpec `json:"overridable,omitempty"`
	Checker          *schema.CheckerConfig `json:"checker,omitempty"`
	Guard            *schema.GuardConfig   `json:"guard,omitempty"`
	Waived           bool                  `json:"waived,omitempty"`
	WaiverIDs        []string              `json:"waiver_ids,omitempty"`
	Provenance       RuleProvenance        `json:"provenance"`
	IsOverlay        bool                  `json:"-"`
}

// Conflict describes a blocking structural, constraint, or confirmed semantic conflict.
type Conflict struct {
	Kind    string `json:"kind"` // constraint | duplicate | invariant | locked | guard | semantic
	RuleID  string `json:"rule_id,omitempty"`
	Target  string `json:"target,omitempty"`
	Level   string `json:"level,omitempty"`
	Message string `json:"message"`
}

// ExceptionView describes an active exception in the resolved view.
type ExceptionView struct {
	ID         string   `json:"id"`
	RuleID     string   `json:"rule_id"`
	Scope      []string `json:"scope"`
	Kind       string   `json:"kind"`
	Expires    string   `json:"expires"`
	ApprovedBy []string `json:"approved_by"`
}

// BudgetInfo tracks rendered size against target limits (§9.7).
type BudgetInfo struct {
	Characters int                    `json:"characters"`
	Targets    map[string]TargetLimit `json:"targets"`
}

// TargetLimit records the character limit for a specific agent target.
type TargetLimit struct {
	Limit int `json:"limit"`
}
