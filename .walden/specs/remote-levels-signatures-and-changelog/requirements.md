---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-23T11:00:37Z
last_modified: 2026-11-23T11:00:37Z
approved_fingerprint: sha256:b08039619feee2ca10b577a31230bc6c2dd5961ae73baa9bb70bc27d1d8edf35
---

# Requirements Document

## Introduction

Phase 2 ("Business unit") hardens remote locked levels and their upgrade lifecycle in Gradus (`docs/design/gradus-design.md` §8, §16, §24.1, §25 X-14, D-09). This feature adds Git signature verification (`signature: required | optional`) recorded in `gradus.lock`, minimum version enforcement (`min_version`) for pinned remote levels, and a readable semantic diff command (`gradusctl changelog`) for lock-update pull requests.

## Requirements

### R1 Git Signature Verification for Remote Levels

**User Story:** As a security or platform lead, I want Gradus to verify Git signatures on locked remote levels and record verification state in `gradus.lock`, so that tampered or unsigned upstream rule releases are rejected before compilation.

#### Acceptance Criteria

1. `R1.AC1` WHEN a level in `gradus.yaml` or its `level.yaml` declares `signature: required` on a git-backed source, the system SHALL verify the resolved tag or commit via `os/exec` `git verify-tag` (for annotated tags) or `git verify-commit` (otherwise) and record `signature: verified` in `gradus.lock`.
   - Acceptance check: Resolving a signed tag or signed commit with `signature: required` succeeds and writes `signature: verified` to `gradus.lock`.
2. `R1.AC2` IF a git-backed level declares `signature: required` and the resolved ref is unsigned or fails Git signature verification, THEN `gradusctl resolve` and `gradusctl verify` SHALL fail with exit code `2` and a clear diagnostic identifying the level ID and verification failure.
   - Acceptance check: Resolving or verifying an unsigned or bad-signature git ref with `signature: required` exits with code `2` and does not write a passing lock or verification verdict.
3. `R1.AC3` WHEN a git-backed level declares `signature: optional` (or omits `signature`), the system SHALL attempt best-effort signature verification, recording `signature: verified` if valid or `signature: none` if unsigned without failing resolution.
   - Acceptance check: An unsigned git level with default/optional signature policy resolves cleanly with `signature: none` in `gradus.lock`.
4. `R1.AC4` IF `signature` is set to an invalid value (anything other than `required` or `optional`), THEN configuration parsing SHALL fail with exit code `2`.
   - Acceptance check: Parsing a `gradus.yaml` or `level.yaml` with `signature: strict` returns a validation error and exit code `2`.

### R2 Minimum Version Enforcement (`min_version`)

**User Story:** As a remote level owner, I want to declare a `min_version` constraint so that consuming repositories cannot stay pinned to retired or vulnerable level versions.

#### Acceptance Criteria

1. `R2.AC1` WHEN a level's `level.yaml` or `gradus.yaml` entry declares `min_version` (SemVer string such as `1.2.0` or `v1.2.0`), the system SHALL compare the resolved level version (`level.yaml` `version` or git tag ref) against `min_version` using SemVer ordering during `resolve` and `verify`.
   - Acceptance check: A level pinned at `v1.3.0` (or `version: 1.3.0`) satisfies `min_version: 1.2.0` and passes resolution and verification.
2. `R2.AC2` IF the resolved version of a level is lower than `min_version` or missing/unparseable as SemVer when `min_version` is declared, THEN `gradusctl resolve` and `gradusctl verify` SHALL fail with exit code `2` and report the resolved version alongside the required `min_version`.
   - Acceptance check: Pinning a level at `v1.1.0` when `min_version: 1.2.0` is declared causes `resolve` and `verify` to exit with code `2` naming both versions.
3. `R2.AC3` IF `min_version` itself is not a valid SemVer string, THEN configuration parsing SHALL fail with exit code `2`.
   - Acceptance check: A `level.yaml` or `gradus.yaml` declaring `min_version: not-a-version` fails parsing with exit code `2`.

### R3 Readable Level Changelog (`gradusctl changelog`)

**User Story:** As a developer reviewing a `gradus.lock` update pull request, I want `gradusctl changelog` to produce a human-readable and JSON diff of rule and metadata changes between two versions of a level, so that I can audit governance updates without reading raw Git diffs.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl changelog <level-id> --from <ref1> --to <ref2> [--json]` runs on a git-backed level, the CLI SHALL materialize both refs via `os/exec` `git` in temporary directories without altering the repository working tree and report added rules, removed rules, modified rules (highlighting changed `severity`, `enforcement`, `scope`, `status`, and `text`), and level metadata changes (`version`, `min_version`, `charter`).
   - Acceptance check: Running `gradusctl changelog` between two commits where one rule was added, one modified, and one removed lists all three accurately in both Markdown and `--json` output while leaving `git status` unchanged.
2. `R3.AC2` WHEN `--from` or `--to` is omitted on `gradusctl changelog <level-id>`, the CLI SHALL default `--from` to the level's locked commit in `gradus.lock` and `--to` to the level's configured `source.ref` in `gradus.yaml` (or `HEAD`).
   - Acceptance check: Running `gradusctl changelog team` after updating `source.ref` in `gradus.yaml` (before running `resolve --write`) diffs the locked commit against the newly configured ref.
3. `R3.AC3` IF `<level-id>` does not exist, is not git-backed, or either ref is invalid or starts with `-`, THEN `gradusctl changelog` SHALL exit with code `2` and a clear error message.
   - Acceptance check: Invoking `gradusctl changelog nonexistent` or passing `--from -evil` exits with code `2`.

## Non-Functional Requirements

- `NFR1` **Static Build & Git via `os/exec` (`PRJ-GO-001`):** All signature verification and changelog git operations MUST shell out to the system `git` binary via `os/exec` (no `go-git`, no `libgit2`, `CGO_ENABLED=0`).
- `NFR2` **Argument Injection Safety:** All git refs passed to `verify-tag`, `verify-commit`, `archive`, or `checkout` MUST reject values starting with `-` and use `--` separators where applicable.
- `NFR3` **Deterministic & Non-Destructive Output:** `gradusctl changelog` MUST emit rules sorted deterministically by rule ID and MUST NOT mutate the working tree, index, or cached level checkout.

## Constraints And Dependencies

- `C1` Must preserve `gradus/v1` backward compatibility (`spec/v1/schema/gradus.schema.json`, `level.schema.json`, `lock.schema.json`) so existing repositories without `signature` or `min_version` continue to work identically.
- `C2` Must reuse existing `internal/schema`, `internal/source`, `internal/lock`, and `internal/resolve` packages without adding third-party cryptographic or git libraries.

## Out Of Scope

- Detached non-Git signatures (cosign, minisign) — deferred to a later iteration after Git GPG/SSH signature verification is established.
- Automated GitHub Pull Request creation bots for lock updates (only the `gradusctl changelog` generator is in scope).
- Remote HTTP/API approval flows (§25).
