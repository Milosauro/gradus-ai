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
	Profile       *schema.Profile  `json:"profile,omitempty"`
	Rules         []ResolvedRule   `json:"rules"`
	InactiveRules []ResolvedRule   `json:"inactive_rules,omitempty"`
	Conflicts     []Conflict       `json:"conflicts"`
	Exceptions    []ExceptionView  `json:"exceptions"`
	Budget        BudgetInfo           `json:"budget"`
	Obligations   []ResolvedObligation `json:"obligations,omitempty"`
	Bindings      []ResolvedBinding    `json:"bindings,omitempty"`
	Toolchain     *ResolvedToolchain   `json:"toolchain,omitempty"`
	Warnings      []string             `json:"warnings,omitempty"`
	Notes         []string             `json:"notes,omitempty"`
}

// ResolvedToolchainItem represents a resolved required, recommended, or forbidden toolchain item (§19).
type ResolvedToolchainItem struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Level  string `json:"level"`
	Reason string `json:"reason,omitempty"`
}

// ResolvedToolchain holds the monotonically merged toolchain declarations across the pyramid (§19).
type ResolvedToolchain struct {
	Required    []ResolvedToolchainItem `json:"required,omitempty"`
	Recommended []ResolvedToolchainItem `json:"recommended,omitempty"`
	Forbidden   []ResolvedToolchainItem `json:"forbidden,omitempty"`
}

// ResolvedObligation describes a resolved process obligation and its matching bindings (§19).
type ResolvedObligation struct {
	ID               string            `json:"id"`
	Rule             string            `json:"rule"`
	Level            string            `json:"level"`
	AllowedProviders []string          `json:"allowed_providers,omitempty"`
	Scope            []string          `json:"scope,omitempty"`
	Bindings         []ResolvedBinding `json:"bindings,omitempty"`
	Satisfied        bool              `json:"satisfied"`
}

// ResolvedBinding describes a resolved provider binding for an obligation (§19).
type ResolvedBinding struct {
	Obligation string            `json:"obligation"`
	Provider   string            `json:"provider"`
	Level      string            `json:"level"`
	Requires   map[string]string `json:"requires,omitempty"`
	Scope      []string          `json:"scope,omitempty"`
}

// ViewScope describes the scope of a resolved view.
type ViewScope struct {
	Repository        string  `json:"repository"`
	Path              *string `json:"path"`
	IncludesUserLevel bool    `json:"includes_user_level"`
}

// LevelSummary summarizes a resolved level.
type LevelSummary struct {
	ID      string          `json:"id"`
	Version string          `json:"version,omitempty"`
	Hash    string          `json:"hash"`
	Charter *schema.Charter `json:"charter,omitempty"`
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
	Status           string                `json:"status,omitempty"`
	ReviewBy         string                `json:"review_by,omitempty"`
	Expires          string                `json:"expires,omitempty"`
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
	ID            string   `json:"id"`
	RuleID        string   `json:"rule_id"`
	Scope         []string `json:"scope"`
	Kind          string   `json:"kind"`
	Reason        string   `json:"reason,omitempty"`
	Created       string   `json:"created,omitempty"`
	Expires       string   `json:"expires"`
	PostReviewDue string   `json:"post_review_due,omitempty"`
	ApprovedBy    []string `json:"approved_by"`
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
