---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-24T15:10:10Z
last_modified: 2026-11-24T15:10:10Z
approved_fingerprint: sha256:4cde006c64e91b18e6ace3ae89966ef23bff5dfc0077abd3be896e8f8b0130e6
---

# Requirements Document

## Introduction

This consolidation release (`v0.3.2`) resolves external audit findings and real-world agent trial feedback after `v0.3.1`: fixing the composite GitHub Action for non-Go repositories, closing residual git-hook bypasses in runtime anti-tamper, expanding `internal/check` edge-case coverage, enforcing `max_exception_duration` at `exception request` time, closing two redaction gaps, upgrading maintenance dependencies (`go.yaml.in/yaml/v3`, Node 24 action pins, `PRJ-CI-001` check promotion), and publishing Design 1.1 with explicit specification precedence.

## Requirements

### R1 Runtime Anti-Tamper, Redaction, and Early Exception Validation

**User Story:** As a security lead, I want runtime anti-tamper to cover remaining git-hook bypasses, redaction to cover unspaced mobile numbers and truncated PEM keys, and `exception request` to reject over-long waivers immediately.

#### Acceptance Criteria

1. `R1.AC1` WHEN a shell command invokes `git -c core.hooksPath=...`, `git push --no-verify`, or sets hook-disabling environment variables (`HUSKY=0`, `LEFTHOOK=0`, `SKIP=...`), `gradusctl guard` SHALL classify it as an anti-tamper mutation and apply `anti_tamper.mode` (`ask` or `deny`).
   - Acceptance check: `TestAntiTamperHardeningV032` verifies each command returns `anti_tamper: true`.
2. `R1.AC2` WHEN a write/edit tool targets `lefthook-local.yml` or `.lefthook-local.yml`, `gradusctl guard` SHALL classify the path as protected and apply `anti_tamper.mode`.
   - Acceptance check: `TestAntiTamperHardeningV032` verifies write attempts to both paths return `anti_tamper: true`.
3. `R1.AC3` WHEN `RedactText` processes an unspaced 10-digit Italian mobile number (`3331234567`) or a truncated PEM private key without `-----END ...-----`, `RedactText` SHALL replace it with `[REDACTED]`.
   - Acceptance check: `TestRedactionGapsV032` verifies both inputs are redacted while benign numbers remain intact.
4. `R1.AC4` IF `gradusctl exception request` is invoked with an expiry exceeding the owning level's `max_exception_duration`, THEN the CLI SHALL reject the request with exit code `2` and state the maximum allowed duration.
   - Acceptance check: `TestExceptionRequestMaxDurationV032` verifies a 180-day request on a 14-day level exits with code `2` and creates no exception file.

### R2 Deterministic Check Engine Edge-Case Coverage

**User Story:** As a maintainer, I want `internal/check` edge cases covered by automated tests so the deterministic enforcement engine behaves predictably on real repositories.

#### Acceptance Criteria

1. `R2.AC1` WHEN `internal/check` evaluates binary files, large files, complex glob scopes, and missing or failing external checkers, the engine SHALL handle binary/large/glob files accurately and report missing or failing external checkers as technical errors (exit code `2`).
   - Acceptance check: `TestCheckEngineEdgeCasesV032` passes in `./internal/check`.

### R3 Non-Go GitHub Action, Supply-Chain Maintenance, and `PRJ-CI-001` Promotion

**User Story:** As a repository maintainer, I want the composite GitHub Action to work on non-Go repositories, CI workflows to use current SHA-pinned actions and maintained YAML libraries, and `PRJ-CI-001` to be checked deterministically.

#### Acceptance Criteria

1. `R3.AC1` WHEN `.github/actions/gradus/action.yml` sets up Go, it SHALL reference `${{ github.action_path }}/../../../go.mod` rather than the consumer repository's `go.mod`, with `.github/workflows/ci.yml` exercising the action on a non-Go directory.
   - Acceptance check: `TestPRJCI001CheckAndActionV032` asserts `action.yml` uses `github.action_path` and `ci.yml` contains the non-Go action test job.
2. `R3.AC2` WHEN workflows and composite actions declare `uses:` references, all references SHALL be pinned to 40-character commit SHAs for current releases and enforced by `PRJ-CI-001` as a deterministic `check` rule with passing `Compliant` and `Non-compliant` examples.
   - Acceptance check: `TestPRJCI001CheckAndActionV032` verifies `gradusctl test` and `gradusctl check` pass for `PRJ-CI-001`.
3. `R3.AC3` WHEN Go packages parse YAML, they SHALL import `go.yaml.in/yaml/v3` instead of `gopkg.in/yaml.v3` with a clean `go mod tidy -diff`.
   - Acceptance check: `go mod tidy -diff` exits `0` and `go.mod` references `go.yaml.in/yaml/v3`.

### R4 Design 1.1, Specification Precedence, Phase 2 Plan, and Transparent README

**User Story:** As a user or auditor, I want `docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/plan/phase-2-plan.md`, `README.md`, and `CHANGELOG.md` to be mutually consistent and transparent about what has been validated.

#### Acceptance Criteria

1. `R4.AC1` WHEN reading `docs/design/gradus-design.md`, the document SHALL be Version `1.1` without `"Implementation has not started"`, aligned with ADR 0001, ADR 0004, JSON lessons (§9.8), active Phase 2 sections (§9.11, §19), `v0.3.x` decisions, and an explicit normative precedence rule across JSON schemas, design, and `SPEC.md`.
   - Acceptance check: `TestDesign11AndSpecAlignmentV032` verifies Design 1.1 markers and absence of stale phrases.
2. `R4.AC2` WHEN reading `spec/v1/SPEC.md`, `docs/plan/phase-2-plan.md`, `README.md`, and `CHANGELOG.md`, those documents SHALL state the normative precedence rule, active Phase 2 keys, Phase 2 plan, `v0.3.2` notes, `--by` + `CODEOWNERS`, `adapter verify` managed-block scope, and transparent validation status.
   - Acceptance check: `TestDesign11AndSpecAlignmentV032` verifies all four documentation files.

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static binary with `CGO_ENABLED=0` and invoke `git` via `os/exec` (`PRJ-GO-001`).
2. `NFR2` All repository files SHALL be written in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` No breaking changes to existing `gradus/v1` YAML/JSON schemas or CLI exit codes (`0`/`1`/`2`).
2. `C2` All existing unit and integration tests SHALL continue to pass without regressions.
