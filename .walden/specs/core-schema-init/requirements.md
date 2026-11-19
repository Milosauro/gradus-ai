---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-19T17:22:13Z
last_modified: 2026-11-19T17:22:13Z
approved_fingerprint: sha256:660f966777e8e9aed853659fb6d02260dd3245f9ba3395ed9726bd35629cc63c
---

# Requirements Document

## Introduction

Feature **F1 (`core-schema-init`)** establishes the foundational `gradus/v1` specification, machine-readable JSON Schemas, Go schema parser/validator (`internal/schema`), repository initialization command (`gradusctl init`), and rule inspection command (`gradusctl explain`). It provides the deterministic data model upon which resolution, compilation, verification, and runtime guards will build.

## Requirements

### R1: Open Specification and JSON Schemas (`spec/v1/`)

**User Story:** As a Gradus maintainer or integrator, I want an open, machine-readable `gradus/v1` specification and JSON Schema set, so that CLI behavior, tooling integrations, and user configurations share a single normative contract.

#### Acceptance Criteria

1. `R1.AC1` The system SHALL provide `spec/v1/SPEC.md` documenting the `gradus/v1` schema for `gradus.yaml`, `level.yaml`, `overlay.yaml`, rule Markdown frontmatter, `gradus.lock`, resolved views, evidence records, lessons, and exceptions.
   - Acceptance check: `spec/v1/SPEC.md` exists and covers every `gradus/v1` artifact defined in `docs/design/gradus-design.md` §9.
2. `R1.AC2` The system SHALL provide valid JSON Schema files under `spec/v1/schema/` for `gradus`, `level`, `overlay`, `rule-frontmatter`, `lock`, `resolved-view`, `evidence`, `lesson`, and `exception`.
   - Acceptance check: every `.schema.json` file under `spec/v1/schema/` parses as valid JSON Schema and validates canonical `gradus/v1` fixture documents.

### R2: Pyramid, Level, and Overlay Configuration Parsing and Validation

**User Story:** As a developer or platform engineer, I want `gradusctl` to parse and validate `gradus.yaml`, `level.yaml`, and `overlay.yaml` deterministically, so that configuration mistakes are caught immediately while forward-compatible keys remain non-breaking.

#### Acceptance Criteria

1. `R2.AC1` WHEN `internal/schema` parses a valid `gradus.yaml` (`schema: gradus/v1`, non-empty ordered `levels` with unique IDs matching `^[a-z][a-z0-9-]*$`), the system SHALL return the typed pyramid configuration with zero errors.
   - Acceptance check: parsing valid `gradus.yaml` fixtures succeeds with the expected level order, targets, hooks, and evidence configuration.
2. `R2.AC2` IF `gradus.yaml`, `level.yaml`, or `overlay.yaml` contains malformed YAML, an unsupported `schema` value, duplicate level IDs, or a level ID failing `^[a-z][a-z0-9-]*$`, THEN the system SHALL return a descriptive validation error that maps to CLI exit code `2`.
   - Acceptance check: invalid configuration fixtures produce non-empty diagnostic errors and never succeed.
3. `R2.AC3` WHEN `gradus.yaml`, `level.yaml`, or `overlay.yaml` contains unknown YAML keys, the system SHALL record each unknown key as a diagnostic note without failing validation.
   - Acceptance check: fixtures with unknown keys parse with zero errors and populate the diagnostic `notes` list with the unknown key paths.
4. `R2.AC4` WHEN `internal/schema` parses `level.yaml` or `overlay.yaml`, the system SHALL validate required metadata (`schema`, `id`) and optional governance sections (`owners`, `approvers`, `charter`, `constraints`, `exceptions`, and overlay `strength`/`activation`).
   - Acceptance check: valid `level.yaml` and `overlay.yaml` fixtures populate all typed fields, and missing `id` or invalid `strength` fails validation.

### R3: Rule Markdown Parsing and Strict Level Derivation

**User Story:** As a rule author, I want each rule stored as a Markdown file with YAML frontmatter whose effective level comes strictly from its directory location, so that rule authoring is human-readable and promotions never break traceability.

#### Acceptance Criteria

1. `R3.AC1` WHEN `internal/schema` parses a valid rule Markdown file, the system SHALL extract frontmatter fields (`id` matching `^[A-Z][A-Z0-9]*(-[A-Z0-9]+)+$`, `title`, `severity` in `MUST|SHOULD|MAY`, `enforcement` defaulting to `prompt`, `scope` defaulting to `["**/*"]`, `overrides`, `overridable`, `checker`, `guard`, `tags`, `version`, `status` defaulting to `active`) and Markdown body sections (statement paragraph, `Why`, `Rationalizations`, and `Examples` with `Compliant`/`Non-compliant`).
   - Acceptance check: parsing a complete rule fixture extracts all frontmatter fields, defaults, statement text, rationalizations, and compliant/non-compliant example blocks.
2. `R3.AC2` IF a rule file is missing YAML frontmatter, misses `id`/`title`/`severity`/statement, uses an invalid `id`/`severity`/`enforcement`, or declares `enforcement: check` without `checker` or `enforcement: hook` without `guard`, THEN the system SHALL reject the file with a validation error.
   - Acceptance check: each malformed rule fixture fails parsing with a specific field-level error message.
3. `R3.AC3` The system SHALL derive a rule's effective `level` exclusively from its containing level container rather than its `id` prefix or frontmatter.
   - Acceptance check: a rule with ID `ORG-SEC-004` loaded inside a `team` level reports `level: "team"`.
4. `R3.AC4` WHEN a rule frontmatter contains unknown YAML keys, the system SHALL record each unknown key as a diagnostic note without failing validation.
   - Acceptance check: a rule fixture with unknown frontmatter keys parses without error and includes the unknown key names in `notes`.

### R4: Repository Initialization (`gradusctl init`)

**User Story:** As a developer starting with Gradus, I want `gradusctl init` to scaffold a working personal/project pyramid in my repository, so that I can adopt Gradus with minimal setup without clobbering existing files.

#### Acceptance Criteria

1. `R4.AC1` WHEN `gradusctl init` runs in a directory without `gradus.yaml`, the system SHALL create `gradus.yaml` (`schema: gradus/v1`), `.gradus/levels/project/level.yaml`, an example rule in `.gradus/levels/project/rules/`, and append missing Gradus git-ignore patterns (`.gradus/cache/`, `.gradus/evidence/`, `CLAUDE.local.md`, `.agents/rules/gradus-user-*.md`) to `.gitignore` while preserving existing `.gitignore` lines.
   - Acceptance check: running `gradusctl init` in a fresh directory with an existing `.gitignore` creates the expected Gradus files and retains all prior `.gitignore` entries without duplicates.
2. `R4.AC2` IF `gradus.yaml` already exists and `--force` is not specified, THEN `gradusctl init` SHALL leave existing files untouched and exit with code `2`.
   - Acceptance check: running `gradusctl init` twice without `--force` returns exit code `2` on the second run and does not overwrite `gradus.yaml`.
3. `R4.AC3` WHEN `gradusctl init --json` succeeds, the system SHALL write a valid JSON summary containing the created and updated file paths to stdout and exit with code `0`.
   - Acceptance check: `gradusctl init --json` exits `0` and outputs parseable JSON listing created/updated paths.

### R5: Rule Inspection (`gradusctl explain <id>`)

**User Story:** As a developer or reviewer, I want to inspect any rule by ID using `gradusctl explain`, so that I can immediately see its origin, level, enforcement, rationale, and examples.

#### Acceptance Criteria

1. `R5.AC1` WHEN `gradusctl explain <id>` is run for a rule present in local configured levels, the system SHALL print the rule ID, title, derived level, source, severity, enforcement, scope, statement, rationale, rejected rationalizations, examples, and `overrides` target if set, and exit with code `0`.
   - Acceptance check: running `gradusctl explain` on a seeded rule outputs all populated metadata and body sections and exits `0`.
2. `R5.AC2` WHEN `gradusctl explain <id> --json` is run for an existing rule, the system SHALL emit structured JSON containing the rule metadata, derived level, source provenance, body fields, and any diagnostic notes, and exit with code `0`.
   - Acceptance check: `gradusctl explain <id> --json` produces valid JSON matching the rule inspection contract.
3. `R5.AC3` IF `<id>` is not found in any configured local level OR the repository configuration is missing/invalid, THEN `gradusctl explain <id>` SHALL print a diagnostic error to stderr and exit with code `2`.
   - Acceptance check: querying an unknown rule ID or running outside a valid Gradus repository exits with code `2`.

## Non-Functional Requirements

- `NFR1` **Zero CGO & Cross-Platform Paths:** All schema parsing, `init`, and `explain` operations MUST compile with `CGO_ENABLED=0` and handle both POSIX (`/`) and Windows (`\`) filesystem paths cleanly.
- `NFR2` **Canonical English:** All generated files, error messages, CLI help text, and specifications MUST be written in English.
- `NFR3` **Exit Code Determinism (D-73):** Configuration, schema, and lookup errors MUST strictly return exit code `2`, never `0`.

## Constraints And Dependencies

- `C1` Uses `github.com/spf13/cobra` for CLI routing and `gopkg.in/yaml.v3` for YAML/frontmatter parsing.
- `C2` Implements ADR 0004 (strict level derivation from container location) and `docs/design/gradus-design.md` §9.1–§9.5.

## Out Of Scope

- Fetching remote `git` sources, `.gradus/cache/`, `gradus.lock` hashing, and full §10 pyramid resolution/conflict detection (covered in **F2: `resolve-and-lock`**).
- Generating agent files (`AGENTS.md`, `CLAUDE.md`, `.agents/rules/`) (covered in **F3: `agent-compilers`**).
- Executing `verify`, `check`, `evidence`, `hooks`, or `guard` (covered in **F4** and **F5**).
