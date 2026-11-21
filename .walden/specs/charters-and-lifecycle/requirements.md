---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T15:39:28Z
last_modified: 2026-11-21T15:39:28Z
approved_fingerprint: sha256:7af31cde0430f1f78c61cc2f7b03abfbec4cba2f29cbf2fca727429f53808db2
---

# Requirements Document

## Introduction

`charters-and-lifecycle` (`F9`) implements deterministic level charter enforcement, rule lifecycle management (`status`, `review_by`, `expires`), and per-level context budgets (`max_rules`, `max_chars`) across `gradusctl resolve`, `gradusctl verify`, `gradusctl compile`, `gradusctl explain`, and the `gradus-author` / `gradus-review` skills.

## Requirements

### R1 Charter Severity Enforcement

**User Story:** As a team or platform owner, I want each level's `charter.allowed_severities` enforced deterministically during resolution and verification, so that lower or narrower levels cannot introduce severities outside their mandate.

#### Acceptance Criteria

1. `R1.AC1` WHEN a level's `level.yaml` defines a non-empty `charter.allowed_severities` list and an active rule in that level declares a `severity` not present in `allowed_severities`, `gradusctl resolve` and `gradusctl verify` SHALL report a charter violation finding and exit with code `1`.
   - Acceptance check: A rule with `severity: MUST` inside a level whose `charter.allowed_severities` is `[SHOULD, MAY]` causes `resolve` and `verify` to emit a finding citing the rule ID, level ID, and allowed severities, and return exit code `1`.
2. `R1.AC2` WHEN a level omits `charter` or omits `charter.allowed_severities`, the system SHALL allow all valid severities (`MUST`, `SHOULD`, `MAY`) for backward compatibility with Phase 0 pyramids.
   - Acceptance check: An existing Phase 0 repository without `charter.allowed_severities` passes `resolve` and `verify` with exit code `0`.

### R2 Charter Metadata Visibility and Skill Guidance

**User Story:** As a developer or reviewer, I want `gradusctl` and the authoring/review skills to surface each level's charter (`purpose`, `includes`, `excludes`, `examples_in`, `examples_out`), so that rules are written at and reviewed against the intended level.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradusctl resolve` builds the resolved view or `gradusctl explain <rule-id>` inspects a rule whose level defines a `charter`, the JSON and human-readable outputs SHALL include the level's charter metadata.
   - Acceptance check: `gradusctl explain <id> --json` and `gradusctl resolve --json` include the level's `charter` fields (`purpose`, `includes`, `excludes`) when defined in `level.yaml`.
2. `R2.AC2` WHEN `gradus-author` or `gradus-review` is packaged in `skills/`, its instructions SHALL reference `charter.includes`, `charter.excludes`, `charter.examples_in`, `charter.examples_out`, and `charter.allowed_severities` when classifying or reviewing rule placement.
   - Acceptance check: `skills/gradus-author/SKILL.md` and `skills/gradus-review/SKILL.md` document charter-based level placement and misplaced-rule checks.

### R3 Rule Lifecycle Status and Review/Expiry Dates

**User Story:** As a governance maintainer, I want rule lifecycle statuses (`draft`, `active`, `deprecated`, `retired`) and `review_by` / `expires` dates validated and handled consistently, so that unfinished drafts or retired rules are not compiled as active rules and overdue or expired rules are surfaced clearly.

#### Acceptance Criteria

1. `R3.AC1` WHEN a rule has `status: draft` or `status: retired`, `gradusctl compile`, `gradusctl check`, and `gradusctl guard` SHALL exclude it from active enforcement and compiled agent instructions while `gradusctl resolve` retains it in lifecycle metadata.
   - Acceptance check: Compiling a pyramid containing a `draft` rule and a `retired` rule produces no entries for those rules in `AGENTS.md`, `CLAUDE.md`, `.claude/rules/`, `.agents/rules/`, and `guard` does not enforce them.
2. `R3.AC2` WHEN a rule has `status: deprecated`, `gradusctl compile` and `gradusctl verify` SHALL keep it active (unless expired) and emit a non-fatal deprecation warning in `verify` output.
   - Acceptance check: A `deprecated` rule remains compiled and produces a warning citing the rule ID without failing `verify` (`exit 0` if no other violations exist).
3. `R3.AC3` WHEN a rule declares `review_by` or `expires`, the system SHALL validate `YYYY-MM-DD` format during schema validation (exit `2` on invalid date), emit a warning in `gradusctl verify` when `review_by` is in the past, and fail `gradusctl verify` with exit `1` when an active rule's `expires` date has passed.
   - Acceptance check: An invalid `review_by: not-a-date` fails schema validation with exit `2`; a past `review_by` emits an overdue-review warning with exit `0`; a past `expires` on an active rule fails `verify` with exit `1`.

### R4 Per-Level Context Budget

**User Story:** As a team lead, I want an optional `context_budget` (`max_rules`, `max_chars`) on each level in `level.yaml`, so that prompt/instruction bloat is caught deterministically before reaching AI agent context windows.

#### Acceptance Criteria

1. `R4.AC1` WHEN `level.yaml` specifies `context_budget.max_rules` and the number of active rules in that level exceeds `max_rules`, `gradusctl resolve` and `gradusctl verify` SHALL report a budget violation and exit with code `1`.
   - Acceptance check: A level with `context_budget.max_rules: 2` and `3` active rules fails `resolve` and `verify` with exit code `1` and cites the level ID, actual count, and limit.
2. `R4.AC2` WHEN `level.yaml` specifies `context_budget.max_chars` and the total rendered character count of active rules in that level exceeds `max_chars`, `gradusctl resolve` and `gradusctl verify` SHALL report a budget violation and exit with code `1`.
   - Acceptance check: A level with `context_budget.max_chars: 100` whose active rules render to `> 100` characters fails `resolve` and `verify` with exit code `1` and cites the level ID, actual character count, and limit.
3. `R4.AC3` WHEN `context_budget` is omitted from `level.yaml`, no budget limit SHALL be imposed on that level.
   - Acceptance check: Existing levels without `context_budget` pass `resolve` and `verify` with exit code `0`.

## Non-Functional Requirements

- `NFR1` **Backward Compatibility:** All new fields (`expires` on rules, `context_budget` on levels) MUST be optional and additive to `gradus/v1` so all Phase 0 configurations and tests continue to pass unchanged.
- `NFR2` **Deterministic Date Evaluation:** Lifecycle date comparisons (`review_by`, `expires`) MUST support an injectable clock in library code so unit tests remain deterministic across timezones and dates.
- `NFR3` **Canonical English & Static Build:** All repository code, schema files, skill updates, and CLI messages MUST be written in canonical English (`PRJ-LANG-001`) and compile with `CGO_ENABLED=0` (`PRJ-GO-001`).

## Constraints And Dependencies

- `C1` JSON Schemas in `spec/v1/schema/` (`level.schema.json`, `rule-frontmatter.schema.json`, `resolved-view.schema.json`) and Go structs in `internal/schema/types.go` / `internal/resolve/types.go` MUST stay synchronized and pass `spec/v1/schema_test.go`.
- `C2` `gradusctl guard` and `gradusctl status --brief` MUST remain `< 100ms` with zero network calls.

## Out Of Scope

- Semantic NLP classification of `charter.includes`/`charter.excludes` in `gradusctl` CLI (handled by skills such as `gradus-author` and `gradus-review`).
- Explicit overlay activation and exception/break-glass workflows (covered in `F10` `overlays-and-exceptions`).
