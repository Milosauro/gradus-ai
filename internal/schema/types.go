package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SchemaVersion is the supported Gradus configuration schema version.
const SchemaVersion = "gradus/v1"

// ParseResult wraps a parsed configuration or rule together with non-fatal diagnostic notes.
type ParseResult[T any] struct {
	Value T
	Notes []string
}

// AntiTamperConfig configures built-in anti-tamper guard behavior (ask | deny).
type AntiTamperConfig struct {
	Mode string `yaml:"mode,omitempty" json:"mode,omitempty"`
}

// Obligation declares an abstract governance requirement satisfied by a provider binding (§19).
type Obligation struct {
	ID               string   `yaml:"id" json:"id"`
	Rule             string   `yaml:"rule" json:"rule"`
	AllowedProviders []string `yaml:"allowed_providers,omitempty" json:"allowed_providers,omitempty"`
	Scope            []string `yaml:"scope,omitempty" json:"scope,omitempty"`
}

// Binding declares how an abstract obligation is satisfied in a scope (§19).
type Binding struct {
	Obligation string            `yaml:"obligation" json:"obligation"`
	Provider   string            `yaml:"provider" json:"provider"`
	Requires   map[string]string `yaml:"requires,omitempty" json:"requires,omitempty"`
	Scope      []string          `yaml:"scope,omitempty" json:"scope,omitempty"`
}

// ToolchainItem declares a required, recommended, or forbidden skill, tool, MCP server, or plugin (§19).
type ToolchainItem struct {
	Type   string `yaml:"type" json:"type"` // skill | tool | mcp | plugin
	ID     string `yaml:"id" json:"id"`
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// ToolchainConfig declares toolchain governance items for a level or pyramid (§19).
type ToolchainConfig struct {
	Required    []ToolchainItem `yaml:"required,omitempty" json:"required,omitempty"`
	Recommended []ToolchainItem `yaml:"recommended,omitempty" json:"recommended,omitempty"`
	Forbidden   []ToolchainItem `yaml:"forbidden,omitempty" json:"forbidden,omitempty"`
}

// PyramidConfig represents gradus.yaml.
type PyramidConfig struct {
	Schema      string            `yaml:"schema" json:"schema"`
	Extends     *string           `yaml:"extends,omitempty" json:"extends,omitempty"`
	AntiTamper  *AntiTamperConfig `yaml:"anti_tamper,omitempty" json:"anti_tamper,omitempty"`
	Levels      []LevelRef        `yaml:"levels" json:"levels"`
	Overlays    []OverlayRef      `yaml:"overlays,omitempty" json:"overlays,omitempty"`
	Profile     *Profile          `yaml:"profile,omitempty" json:"profile,omitempty"`
	Targets     []string          `yaml:"targets,omitempty" json:"targets,omitempty"`
	Hooks       *HooksConfig      `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	Evidence    *EvidenceConfig   `yaml:"evidence,omitempty" json:"evidence,omitempty"`
	Obligations []Obligation      `yaml:"obligations,omitempty" json:"obligations,omitempty"`
	Bindings    []Binding         `yaml:"bindings,omitempty" json:"bindings,omitempty"`
	Toolchain   *ToolchainConfig  `yaml:"toolchain,omitempty" json:"toolchain,omitempty"`
}

// LevelRef references a pyramid level in gradus.yaml.
type LevelRef struct {
	ID            string         `yaml:"id" json:"id"`
	Source        SourceRef      `yaml:"source" json:"source"`
	Locked        bool           `yaml:"locked,omitempty" json:"locked,omitempty"`
	Signature     string         `yaml:"signature,omitempty" json:"signature,omitempty"`
	MinVersion    string         `yaml:"min_version,omitempty" json:"min_version,omitempty"`
	Offline       *OfflinePolicy `yaml:"offline,omitempty" json:"offline,omitempty"`
	Approvers     []string       `yaml:"approvers,omitempty" json:"approvers,omitempty"`
	EditableRules []string       `yaml:"editable_rules,omitempty" json:"editable_rules,omitempty"`
	Scope         []string       `yaml:"scope,omitempty" json:"scope,omitempty"`
}

// SourceRef specifies a local path or git repository source.
type SourceRef struct {
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
	Git  string `yaml:"git,omitempty" json:"git,omitempty"`
	Ref  string `yaml:"ref,omitempty" json:"ref,omitempty"`
}

// OfflinePolicy defines how unreachable remote sources behave.
type OfflinePolicy struct {
	Mode   string `yaml:"mode" json:"mode"`
	MaxAge string `yaml:"max_age,omitempty" json:"max_age,omitempty"`
}

// OverlayRef references an overlay in gradus.yaml.
type OverlayRef struct {
	ID         string    `yaml:"id" json:"id"`
	Source     SourceRef `yaml:"source" json:"source"`
	Strength   string    `yaml:"strength" json:"strength"`
	Activation string    `yaml:"activation,omitempty" json:"activation,omitempty"`
}

// Profile declares project applicability facts for conditional overlays (D-62).
type Profile struct {
	Data     string         `yaml:"data,omitempty" json:"data,omitempty"`
	Exposure string         `yaml:"exposure,omitempty" json:"exposure,omitempty"`
	Domain   []string       `yaml:"domain,omitempty" json:"domain,omitempty"`
	Client   []string       `yaml:"client,omitempty" json:"client,omitempty"`
	Extra    map[string]any `yaml:",inline" json:"-"`
}

// MarshalJSON flattens standard and custom Extra fields into a single JSON object.
func (p Profile) MarshalJSON() ([]byte, error) {
	m := make(map[string]any, len(p.Extra)+4)
	for k, v := range p.Extra {
		m[k] = v
	}
	if p.Data != "" {
		m["data"] = p.Data
	}
	if p.Exposure != "" {
		m["exposure"] = p.Exposure
	}
	if len(p.Domain) > 0 {
		m["domain"] = p.Domain
	}
	if len(p.Client) > 0 {
		m["client"] = p.Client
	}
	return json.Marshal(m)
}

// UnmarshalJSON decodes standard fields and preserves unknown keys in Extra.
func (p *Profile) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Extra = nil
	for k, v := range raw {
		switch k {
		case "data":
			if err := json.Unmarshal(v, &p.Data); err != nil {
				return err
			}
		case "exposure":
			if err := json.Unmarshal(v, &p.Exposure); err != nil {
				return err
			}
		case "domain":
			if err := json.Unmarshal(v, &p.Domain); err != nil {
				return err
			}
		case "client":
			if err := json.Unmarshal(v, &p.Client); err != nil {
				return err
			}
		default:
			var val any
			if err := json.Unmarshal(v, &val); err != nil {
				return err
			}
			if p.Extra == nil {
				p.Extra = make(map[string]any)
			}
			p.Extra[k] = val
		}
	}
	return nil
}

// Lookup returns normalized string values for a standard or custom profile attribute.
func (p *Profile) Lookup(key string) ([]string, bool) {
	if p == nil {
		return nil, false
	}
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "data":
		if strings.TrimSpace(p.Data) == "" {
			return nil, false
		}
		return []string{strings.TrimSpace(p.Data)}, true
	case "exposure":
		if strings.TrimSpace(p.Exposure) == "" {
			return nil, false
		}
		return []string{strings.TrimSpace(p.Exposure)}, true
	case "domain":
		if len(p.Domain) == 0 {
			return nil, false
		}
		return p.Domain, true
	case "client":
		if len(p.Client) == 0 {
			return nil, false
		}
		return p.Client, true
	}
	for k, v := range p.Extra {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			vals := NormalizeProfileValues(v)
			if len(vals) == 0 {
				return nil, false
			}
			return vals, true
		}
	}
	return nil, false
}

// NormalizeProfileValues flattens scalar or slice values into trimmed string values.
func NormalizeProfileValues(v any) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if s := strings.TrimSpace(t); s != "" {
			return []string{s}
		}
		return nil
	case []string:
		var out []string
		for _, s := range t {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		var out []string
		for _, item := range t {
			out = append(out, NormalizeProfileValues(item)...)
		}
		return out
	default:
		if s := strings.TrimSpace(fmt.Sprintf("%v", t)); s != "" {
			return []string{s}
		}
		return nil
	}
}

// ActivationRule defines a profile-driven overlay activation rule in level.yaml (§9.3).
type ActivationRule struct {
	Overlay  string         `yaml:"overlay" json:"overlay"`
	When     map[string]any `yaml:"when" json:"when"`
	Required bool           `yaml:"required,omitempty" json:"required,omitempty"`
}

// HooksConfig configures git hook manager integration.
type HooksConfig struct {
	Manager string `yaml:"manager" json:"manager"`
}

// EvidenceConfig configures evidence sinks for local and CI execution (§12).
type EvidenceConfig struct {
	Sinks []SinkConfig       `yaml:"sinks,omitempty" json:"sinks,omitempty"`
	CI    *EvidenceSinksOnly `yaml:"ci,omitempty" json:"ci,omitempty"`
}

// EvidenceSinksOnly holds CI-specific evidence sinks (§12).
type EvidenceSinksOnly struct {
	Sinks []SinkConfig `yaml:"sinks,omitempty" json:"sinks,omitempty"`
}

// SinkConfig represents an evidence destination.
type SinkConfig struct {
	Type      string `yaml:"type" json:"type"`
	Path      string `yaml:"path,omitempty" json:"path,omitempty"`
	Command   string `yaml:"command,omitempty" json:"command,omitempty"`
	Retention string `yaml:"retention,omitempty" json:"retention,omitempty"`
}

// LevelConfig represents level.yaml.
type LevelConfig struct {
	Schema          string           `yaml:"schema" json:"schema"`
	ID              string           `yaml:"id" json:"id"`
	Version         string           `yaml:"version,omitempty" json:"version,omitempty"`
	MinVersion      string           `yaml:"min_version,omitempty" json:"min_version,omitempty"`
	Signature       string           `yaml:"signature,omitempty" json:"signature,omitempty"`
	Owners          []string         `yaml:"owners,omitempty" json:"owners,omitempty"`
	Approvers       []string         `yaml:"approvers,omitempty" json:"approvers,omitempty"`
	Charter         *Charter         `yaml:"charter,omitempty" json:"charter,omitempty"`
	ContextBudget   *ContextBudget   `yaml:"context_budget,omitempty" json:"context_budget,omitempty"`
	Constraints     *Constraints     `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	ActivationRules []ActivationRule `yaml:"activation_rules,omitempty" json:"activation_rules,omitempty"`
	Exceptions      *ExceptionPolicy `yaml:"exceptions,omitempty" json:"exceptions,omitempty"`
	Obligations     []Obligation     `yaml:"obligations,omitempty" json:"obligations,omitempty"`
	Bindings        []Binding        `yaml:"bindings,omitempty" json:"bindings,omitempty"`
	Toolchain       *ToolchainConfig `yaml:"toolchain,omitempty" json:"toolchain,omitempty"`
}

// ContextBudget defines optional per-level context limits (X-07).
type ContextBudget struct {
	MaxRules *int `yaml:"max_rules,omitempty" json:"max_rules,omitempty"`
	MaxChars *int `yaml:"max_chars,omitempty" json:"max_chars,omitempty"`
}

// Charter defines what belongs to a level.
type Charter struct {
	Purpose           string   `yaml:"purpose,omitempty" json:"purpose,omitempty"`
	Includes          []string `yaml:"includes,omitempty" json:"includes,omitempty"`
	Excludes          []string `yaml:"excludes,omitempty" json:"excludes,omitempty"`
	ExamplesIn        []string `yaml:"examples_in,omitempty" json:"examples_in,omitempty"`
	ExamplesOut       []string `yaml:"examples_out,omitempty" json:"examples_out,omitempty"`
	AllowedSeverities []string `yaml:"allowed_severities,omitempty" json:"allowed_severities,omitempty"`
	ReviewCadence     string   `yaml:"review_cadence,omitempty" json:"review_cadence,omitempty"`
}

// Constraints defines inherited constraints placed on lower levels.
type Constraints struct {
	RequireLevels   []string `yaml:"require_levels,omitempty" json:"require_levels,omitempty"`
	MaxExtraLevels  *int     `yaml:"max_extra_levels,omitempty" json:"max_extra_levels,omitempty"`
	AllowedOverlays []string `yaml:"allowed_overlays,omitempty" json:"allowed_overlays,omitempty"`
}

// ExceptionPolicy defines exception limits for a level.
type ExceptionPolicy struct {
	MaxDuration string   `yaml:"max_duration,omitempty" json:"max_duration,omitempty"`
	Approvers   []string `yaml:"approvers,omitempty" json:"approvers,omitempty"`
	BreakGlass  bool     `yaml:"break_glass,omitempty" json:"break_glass,omitempty"`
}

// OverlayConfig represents overlay.yaml.
type OverlayConfig struct {
	Schema     string   `yaml:"schema" json:"schema"`
	ID         string   `yaml:"id" json:"id"`
	Version    string   `yaml:"version,omitempty" json:"version,omitempty"`
	Strength   string   `yaml:"strength" json:"strength"`
	Owners     []string `yaml:"owners,omitempty" json:"owners,omitempty"`
	Approvers  []string `yaml:"approvers,omitempty" json:"approvers,omitempty"`
	Activation string         `yaml:"activation,omitempty" json:"activation,omitempty"`
	When       map[string]any `yaml:"when,omitempty" json:"when,omitempty"`
}

// OverridableSpec supports either a boolean or an object with override constraints (§9.4).
type OverridableSpec struct {
	Allowed     bool           `json:"allowed"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// UnmarshalYAML decodes either a YAML boolean or a YAML mapping into OverridableSpec.
func (o *OverridableSpec) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var b bool
		if err := value.Decode(&b); err != nil {
			return fmt.Errorf("overridable scalar must be a boolean: %w", err)
		}
		o.Allowed = b
		o.Constraints = nil
		return nil
	case yaml.MappingNode:
		var m map[string]any
		if err := value.Decode(&m); err != nil {
			return fmt.Errorf("overridable mapping invalid: %w", err)
		}
		o.Allowed = true
		o.Constraints = m
		return nil
	default:
		return fmt.Errorf("overridable must be a boolean or mapping, got YAML node kind %d", value.Kind)
	}
}

// CheckerConfig configures deterministic code checks for enforcement: check.
type CheckerConfig struct {
	Name    string `yaml:"name" json:"name"`
	Config  string `yaml:"config,omitempty" json:"config,omitempty"`
	Pattern string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
}

// GuardConfig configures runtime agent action guards for enforcement: hook.
type GuardConfig struct {
	Action   string `yaml:"action" json:"action"`
	Pattern  string `yaml:"pattern" json:"pattern"`
	Decision string `yaml:"decision" json:"decision"`
}

// ExamplePair holds compliant and non-compliant code examples from a rule body.
type ExamplePair struct {
	Compliant    []string `json:"compliant,omitempty"`
	NonCompliant []string `json:"non_compliant,omitempty"`
}

// Rule represents a parsed rule Markdown file and its derived metadata.
type Rule struct {
	ID              string           `yaml:"id" json:"id"`
	Title           string           `yaml:"title" json:"title"`
	Level           string           `yaml:"-" json:"level"` // Derived strictly from container (ADR 0004)
	Severity        string           `yaml:"severity" json:"severity"`
	Enforcement     string           `yaml:"enforcement,omitempty" json:"enforcement"`
	Scope           []string         `yaml:"scope,omitempty" json:"scope"`
	Overrides       string           `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Overridable     *OverridableSpec `yaml:"overridable,omitempty" json:"overridable,omitempty"`
	AppliesWhen     map[string]any   `yaml:"applies_when,omitempty" json:"applies_when,omitempty"`
	Source          string           `yaml:"source,omitempty" json:"source,omitempty"`
	Controls        []string         `yaml:"controls,omitempty" json:"controls,omitempty"`
	Owner           string           `yaml:"owner,omitempty" json:"owner,omitempty"`
	Status          string           `yaml:"status,omitempty" json:"status"`
	ReviewBy        string           `yaml:"review_by,omitempty" json:"review_by,omitempty"`
	Expires         string           `yaml:"expires,omitempty" json:"expires,omitempty"`
	Checker         *CheckerConfig   `yaml:"checker,omitempty" json:"checker,omitempty"`
	Guard           *GuardConfig     `yaml:"guard,omitempty" json:"guard,omitempty"`
	Tags            []string         `yaml:"tags,omitempty" json:"tags,omitempty"`
	Version         int              `yaml:"version,omitempty" json:"version,omitempty"`
	Statement       string           `yaml:"-" json:"statement"`
	Why             string           `yaml:"-" json:"why,omitempty"`
	Rationalizations []string        `yaml:"-" json:"rationalizations,omitempty"`
	Examples        ExamplePair      `yaml:"-" json:"examples,omitempty"`
	FilePath        string           `yaml:"-" json:"file_path,omitempty"`
	ProvenanceSource string          `yaml:"-" json:"provenance_source,omitempty"`
}
