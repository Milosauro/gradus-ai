---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T15:50:39Z
last_modified: 2026-11-21T15:50:39Z
approved_fingerprint: sha256:723605b7f015b07b41f1639062f4e5f0d23c24fcb931a8b6b4ba5b8382bc5f85
source_requirements_approved_at: 2026-11-21T15:39:28Z
source_requirements_fingerprint: sha256:7af31cde0430f1f78c61cc2f7b03abfbec4cba2f29cbf2fca727429f53808db2
---

# Feature Design

## Architecture

This feature extends the existing schema, resolver, CLI, and embedded skill layers with additive fields and deterministic checks while preserving full Phase 0 backward compatibility.

### 1. Schema Layer (`internal/schema/` & `spec/v1/schema/`)
- **`ContextBudget` struct (`internal/schema/types.go`):** Add optional `ContextBudget *ContextBudget` (`yaml:"context_budget,omitempty" json:"context_budget,omitempty"`) to `LevelConfig`, with `MaxRules *int` (`max_rules`) and `MaxChars *int` (`max_chars`).
- **`Rule.Expires` (`internal/schema/types.go`):** Add optional `Expires string` (`yaml:"expires,omitempty" json:"expires,omitempty"`) alongside existing `Status` and `ReviewBy`.
- **Validation (`internal/schema/validate_rule.go`, `internal/schema/validate_config.go`):**
  - Validate `ReviewBy` and `Expires` (when non-empty) against `time.Parse("2006-01-02", v)`; invalid dates fail schema validation (`ExitError` / `2`).
  - Validate `Charter.AllowedSeverities` entries against `MUST|SHOULD|MAY` and `ContextBudget` values (`>= 0`).
- **JSON Schemas (`spec/v1/schema/`):** Add `context_budget` to `level.schema.json`, `expires` to `rule-frontmatter.schema.json`, and `charter` / `inactive_rules` / `warnings` to `resolved-view.schema.json`.

### 2. Resolution & Policy Enforcement (`internal/resolve/`)
- **Clock injection (`resolve.Options`):** Add optional `Now func() time.Time` (defaults to `time.Now`) for deterministic date evaluation (`NFR2`).
- **ResolvedView extensions (`internal/resolve/types.go`):**
  - `LevelSummary.Charter *schema.Charter` (`json:"charter,omitempty"`) to expose level charter metadata in `resolve --json`.
  - `ResolvedRule` gains `Status`, `ReviewBy`, `Expires`.
  - `ResolvedView.InactiveRules []ResolvedRule` (`json:"inactive_rules,omitempty"`) holds rules with `status: draft` or `status: retired`, keeping `ResolvedView.Rules` strictly active (`active` / `deprecated`).
  - `ResolvedView.Warnings []string` (`json:"warnings,omitempty"`) collects non-fatal lifecycle warnings (`deprecated` rules and overdue `review_by` dates).
- **Deterministic checks in `resolve.Resolve`:**
  1. **Lifecycle partitioning:** Rules with `status == "draft"` or `status == "retired"` are routed to `InactiveRules` so `compile`, `check`, and `guard` automatically exclude them without per-consumer filtering.
  2. **Charter severity check (`R1`):** For each active rule in a level with non-empty `charter.allowed_severities`, verify `rule.Severity` is allowed; otherwise append `Conflict{Kind: "charter", RuleID, Level, Message}`.
  3. **Lifecycle date & deprecation checks (`R3`):** For each active rule:
     - `status == "deprecated"` appends a non-fatal warning to `Warnings`.
     - `review_by != ""` and `review_by < todayUTC` appends a non-fatal overdue-review warning to `Warnings`.
     - `expires != ""` and `expires < todayUTC` appends `Conflict{Kind: "lifecycle", RuleID, Level, Message}` (causing `resolve` / `verify` to exit `1`).
  4. **Per-level context budget (`R4`):** For each level defining `context_budget`:
     - Count active rules in that level; if `MaxRules != nil && count > *MaxRules`, append `Conflict{Kind: "budget", Level, Message}`.
     - Sum rendered character lengths (`len(rule.ID) + len(rule.Title) + len(rule.Text) + len(rule.Why)` or canonical rule markdown block length) for active rules in that level; if `MaxChars != nil && chars > *MaxChars`, append `Conflict{Kind: "budget", Level, Message}`.

### 3. CLI & Skills (`internal/cli/`, `skills/`)
- **`gradusctl verify` (`internal/cli/verify.go`):** Include `Warnings` in JSON and text output; `Conflicts` from `charter`, `lifecycle`, and `budget` already flow through `verify`'s existing conflict check and yield exit code `1`.
- **`gradusctl explain` (`internal/cli/explain.go`):** Look up the rule's owning level in `pyramid.Levels` and include `Charter *schema.Charter` in `explainResult` (both JSON and human-readable output).
- **Embedded skills (`skills/gradus-author/SKILL.md`, `skills/gradus-review/SKILL.md`):** Add explicit charter-guidance sections instructing agents to consult `charter.purpose`, `includes`, `excludes`, `examples_in`, `examples_out`, and `allowed_severities` when authoring or reviewing rules.

## Options Considered

- **Chosen: Partition `draft`/`retired` rules into `ResolvedView.InactiveRules` during resolution.**
  - Every downstream consumer (`compile`, `check`, `guard`) iterates `ResolvedView.Rules` and therefore excludes `draft`/`retired` rules automatically with zero duplicated status checks, while `resolve --json` still exposes `inactive_rules` for transparency.
- **Alternative rejected: Keep `draft`/`retired` inside `ResolvedView.Rules` and filter in every compiler, checker, and guard.**
  - Rejected because forgetting a filter in any current or future consumer (e.g., Phase 1 `handbook` or `test`) would silently enforce draft or retired rules.

## Simplicity And Elegance Review

- Reuses the existing `resolve.Conflict` pipeline (`Kind: "charter" | "lifecycle" | "budget"`) so `gradusctl resolve` and `gradusctl verify` inherit consistent exit codes (`1` on violations, `2` on invalid schema dates) and JSON envelopes with minimal new code.
- Reuses `schema.Charter` already defined in `internal/schema/types.go` and `spec/v1/schema/level.schema.json`.
- Zero new third-party dependencies; date validation uses Go stdlib `time.Parse("2006-01-02", ...)`.

## Failure Modes And Tradeoffs

- **Invalid date format (`review_by` / `expires`):** Fails closed during schema validation with exit `2` rather than silently ignoring malformed dates.
- **Timezone ambiguity on date-only fields:** Comparing `YYYY-MM-DD` strings against `now().UTC().Format("2006-01-02")` ensures deterministic behavior across developer laptops and CI runners.
- **Rendered character budget approximation:** Measuring each active rule's canonical rendered text length (`ID + Title + Statement + Why + Rationalizations`) is fast, deterministic, and target-independent, though individual agent compilers may add small wrapper headers.

## Verification Plan

- Unit tests in `internal/schema` verifying valid/invalid `review_by`, `expires`, `charter.allowed_severities`, and `context_budget`.
- Unit tests in `internal/resolve` (with injected `Now` clock) verifying charter severity violations, `draft`/`retired` exclusion into `InactiveRules`, `deprecated` & overdue `review_by` warnings, expired rule conflict, and `max_rules`/`max_chars` budget enforcement.
- CLI integration tests in `internal/cli` verifying `gradusctl resolve`, `verify`, `compile`, `guard`, and `explain` exit codes and JSON outputs.
- Skill verification test in `skills/skills_test.go` and schema validation test in `spec/v1/schema_test.go`.

## Requirement Coverage

| Requirement | Covered By |
| --- | --- |
| `R1` | `R1.AC1`–`R1.AC2`: `resolve.Resolve` charter `allowed_severities` check emitting `Conflict{Kind: "charter"}` (exit `1`), allowing all severities when omitted |
| `R2` | `R2.AC1`–`R2.AC2`: `LevelSummary.Charter` in `resolve --json`, `explainResult.Charter` in `explain`, and charter guidance in `skills/gradus-author/SKILL.md` and `skills/gradus-review/SKILL.md` |
| `R3` | `R3.AC1`–`R3.AC3`: `draft`/`retired` partitioned into `ResolvedView.InactiveRules`, `deprecated` & overdue `review_by` warnings in `ResolvedView.Warnings`, `YYYY-MM-DD` validation (exit `2`), and expired `expires` conflict (exit `1`) |
| `R4` | `R4.AC1`–`R4.AC3`: `context_budget.max_rules` and `context_budget.max_chars` checks in `resolve.Resolve` emitting `Conflict{Kind: "budget"}` (exit `1`), unlimited when omitted |
| `NFR1` | All new YAML/JSON fields are optional (`omitempty`) and existing Phase 0 tests remain green |
| `NFR2` | Injectable `Options.Now func() time.Time` in `internal/resolve` |
| `NFR3` | Canonical English across all files and `CGO_ENABLED=0` static build |
| `C1` | Synchronized JSON schemas in `spec/v1/schema/` verified by `spec/v1/schema_test.go` |
| `C2` | All charter/lifecycle/budget checks run in-memory offline; `guard` and `status --brief` remain `< 100ms` |
