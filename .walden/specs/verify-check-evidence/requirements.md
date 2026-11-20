---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T09:48:28Z
last_modified: 2026-11-20T09:48:28Z
approved_fingerprint: sha256:dbd6945d30689f079b80581f0d2bfed47321988dfd925c8c6273b26fc2161fd5
---

# Requirements Document

## Introduction

Feature **F4 (`verify-check-evidence`)** implements the universal enforcement and auditability layer (Level 3, §11.1): `gradusctl verify` (lock integrity, conflicts, expired exceptions, and generated-file drift detection), `gradusctl check` (deterministic built-in and external checker plugin execution on `--staged` or `--base <ref>`), `gradusctl evidence` (the 5 configurable sinks matching §9.9), and `gradusctl hooks` (git hook manager detection and configuration).

## Requirements

### R1: Pyramid & Generated-File Verification (`gradusctl verify`)

**User Story:** As a developer or CI pipeline, I want `gradusctl verify` to check pyramid integrity, conflicts, expired exceptions, and generated-file drift, so that no unapproved or out-of-date configuration reaches the main branch.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl verify` runs on a valid, conflict-free pyramid whose `gradus.lock` matches locked levels and whose compiled target files match `PlanCompile` byte-for-byte, the system SHALL exit with code `0`.
   - Acceptance check: running `gradusctl verify --json` after `resolve --write` and `compile` exits `0` with `verdict: "pass"`.
2. `R1.AC2` IF the pyramid has blocking conflicts, expired exceptions, OR any compiled target file on disk differs from `PlanCompile` (drift), THEN `gradusctl verify` SHALL report each finding with file/rule context and exit with code `1`.
   - Acceptance check: manually editing `AGENTS.md` or `.agents/rules/gradus-*.md` causes `gradusctl verify` to report drift and exit `1` (E2).
3. `R1.AC3` IF `gradus.yaml` is invalid, `gradus.lock` is missing or mismatches a `locked: true` level, OR a `fail-closed` source is unreachable, THEN `gradusctl verify` SHALL report the error and exit with code `2`.
   - Acceptance check: tampering a `locked: true` rule without updating `gradus.lock` causes `gradusctl verify` to exit `2` (E2).

### R2: Deterministic Code Checks (`gradusctl check`)

**User Story:** As a security or platform engineer, I want `gradusctl check` to run deterministic pattern checks and external checker plugins on staged or branch changes, so that policy violations are blocked consistently across all agents.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradusctl check` runs with `--staged` or `--base <ref>`, the system SHALL determine changed files in scope using `git diff` via `os/exec` and filter rules with `enforcement: check` whose `scope` globs match the changed files (skipping waived rules).
   - Acceptance check: only changed files matching a rule's `scope` are evaluated, and unchanged files are ignored.
2. `R2.AC2` WHEN a matching `check` rule uses the built-in pattern checker (`checker.name: pattern` or `regex`), the system SHALL scan the changed files and report any match with rule ID, severity, level, source, file path, and line number, exiting with code `1`.
   - Acceptance check: staging a file containing a forbidden pattern triggers a structured finding citing the rule ID and exits `1`.
3. `R2.AC3` WHEN a matching `check` rule specifies an external checker (`checker.name` such as `gitleaks`, `semgrep`, or a custom binary), the system SHALL invoke the external checker on the scoped files and map non-zero violation exits to `1`.
   - Acceptance check: an external checker reporting violations produces structured findings and exits `1`.
4. `R2.AC4` IF a configured external checker binary is missing from `PATH` OR exits with an execution/configuration error, THEN `gradusctl check` SHALL fail with exit code `2` and NEVER report a pass.
   - Acceptance check: referencing a nonexistent checker binary in a matching rule causes `gradusctl check` to exit `2` (D-73).

### R3: Verifiable Evidence Sinks (`gradusctl evidence`)

**User Story:** As a compliance or engineering team, I want `gradusctl` to emit machine-readable evidence bound to the git commit and lock hash into configurable sinks, so that checks are auditable without tracking individual developer activity.

#### Acceptance Criteria

1. `R3.AC1` The system SHALL produce evidence records conforming to `spec/v1/schema/evidence.schema.json` (`schema: gradus/v1`, `id`, `kind`, `timestamp`, `commit`, `lock_hash`, `view_hash`, `producer`, `findings`, `verdict`) with zero personal activity metrics (P16).
   - Acceptance check: generated evidence JSON validates against `evidence.schema.json` and contains `lock_hash`, `view_hash`, and `verdict`.
2. `R3.AC2` WHEN writing evidence, the system SHALL support all 5 sink types (`none`, `local` in `.gradus/evidence/`, `ci-artifact` directory, `repo` committed path, and `command` piped via stdin), selecting `evidence.ci.sinks` when `CI` is set if configured.
   - Acceptance check: unit tests exercise all 5 sink behaviors, including `command` receiving the JSON record on stdin.
3. `R3.AC3` WHEN `gradusctl evidence list` or `gradusctl evidence write` runs, the system SHALL list local `.gradus/evidence/*.json` records (pruning expired records past retention) or write an evidence record to configured sinks.
   - Acceptance check: `gradusctl evidence list --json` returns stored local records sorted newest-first.

### R4: Git Hook Manager Integration (`gradusctl hooks`)

**User Story:** As a developer, I want `gradusctl hooks` to detect my repository's git hook manager and output or install pre-commit hooks, so that `verify` and `check --staged` run automatically before every commit.

#### Acceptance Criteria

1. `R4.AC1` WHEN `gradusctl hooks` runs with `hooks.manager: auto`, the system SHALL detect `.pre-commit-config.yaml` (`pre-commit`), `lefthook.yml` (`lefthook`), `.husky/` (`husky`), or fall back to `native` (`.githooks/`) in that order.
   - Acceptance check: detection returns the matching manager name for each repository fixture.
2. `R4.AC2` WHEN `gradusctl hooks print` or `gradusctl hooks install` runs, the system SHALL emit the manager-specific configuration snippet running `gradusctl verify` and `gradusctl check --staged`, and with `install` write/update the hook config (requiring explicit `--native` consent before modifying `core.hooksPath`).
   - Acceptance check: `hooks install` updates `lefthook.yml` / `.pre-commit-config.yaml` / `.husky/pre-commit` idempotently and refuses `core.hooksPath` mutation without `--native` consent.

## Non-Functional Requirements

- `NFR1` **No Error-as-Pass Guarantee (D-73):** Missing checkers, broken git diff commands, or invalid lockfiles MUST return exit code `2`, never `0`.
- `NFR2` **Subprocess Safety (`os/exec`):** All `git diff`, external checker, and `command` sink invocations MUST use `os/exec` with zero shell interpolation unless explicitly configured in `command` sink.
- `NFR3` **Zero Personal Metrics (P16):** Evidence records MUST contain no user-activity tracking beyond standard git commit metadata.

## Constraints And Dependencies

- `C1` Builds on `internal/resolve` (`F2`), `internal/compile.PlanCompile` (`F3`), and `github.com/bmatcuk/doublestar/v4`.
- `C2` Implements `gradus-design.md` §9.9, §11.4, §11.5, §12, and D-71/D-72/D-73.

## Out Of Scope

- Runtime agent `PreToolUse` evaluation (`gradusctl guard`, covered in **F5: `runtime-guard`**).
- Interactive drift repair (`gradusctl reconcile`, covered in **F6: `cli-lifecycle-state`**).
- Model-based code review (`gradus-review`, covered in **F7: `mvp-skills`**).
