---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-26T12:15:36Z
last_modified: 2026-11-26T12:15:36Z
approved_fingerprint: sha256:78e0c852d44c8eae3d6f01e02d3ac339b84c5bbd31886ac3b89ab11cc0b5d1fe
---

# Requirements Document

## Introduction

Release `v0.4.2` (`F30`, `anti-tamper-and-boundary-hardening`) addresses four concrete post-release findings from the `v0.4.1` audit and live harness tests:

1. Restore anti-tamper detection for hook-bypass environment variables set via `export` or standalone shell assignments (`export HUSKY=0; git commit`, `export SKIP=gradus`, `set -a; HUSKY=0; git commit`) without reintroducing false positives on harmless text arguments (`echo SKIP=foo`, `grep LEFTHOOK=0`, commit messages).
2. Enforce `.git` repository boundaries inside `guard.resolveWorkspaceRoot` so nested independent Git repositories (such as submodules or cloned repositories inside a Gradus workspace or home directory) never inherit a parent directory's `gradus.yaml`.
3. Honor Decision `D-82` in `gradusctl status --hook` and `gradusctl status --antigravity-hook` by exiting `0` with a diagnostic summary when `gradus.yaml` contains a syntax or schema error, while keeping `gradusctl guard` fail-closed (`exit 2`) and documenting manual recovery.
4. Refine the `propose_rule` invalid-level diagnostic message in the MCP server, close open item `O-1` (§24.3) in `docs/design/gradus-design.md` to match the real `wallet-tracker` validation documented in `README.md`, and update release documentation for `v0.4.2`.

## Requirements

### R1 Anti-Tamper Detection for Exported and Standalone Hook-Bypass Variables

**User Story:** As a repository maintainer, I want `gradusctl guard` to catch hook-bypass environment variables set via `export` or standalone shell assignments without false positives on harmless text, so that agents cannot bypass git hooks via multi-statement shell commands.

#### Acceptance Criteria

1. `R1.AC1` WHEN a shell command sets a hook-bypass environment variable (`HUSKY=0`, `LEFTHOOK=0`, `SKIP=...`, `LEFTHOOK_EXCLUDE=...`, or `GIT_CONFIG_COUNT=...`) via `export` or as a standalone statement separated by shell delimiters (`;`, `&&`, `||`, `|`, or newline), the system SHALL trigger anti-tamper confirmation (`ask` or the configured `anti_tamper.mode`).
   - Acceptance check: `export HUSKY=0; git commit -m x`, `export HUSKY=0 && git commit -m x`, `export SKIP=gradus`, and `set -a; HUSKY=0; git commit` each produce an anti-tamper decision rather than `allow`.
2. `R1.AC2` WHEN a shell command contains hook-bypass variable strings only inside non-assignment command arguments such as `echo`, `grep`, or a `git commit -m` message, the system SHALL NOT trigger the hook-bypass anti-tamper check.
   - Acceptance check: `echo SKIP=foo`, `grep LEFTHOOK=0 README.md`, and `git commit -m "document HUSKY=0"` do not trigger the hook-bypass anti-tamper rule.

### R2 Nested Git Repository Boundary Isolation

**User Story:** As a developer working with nested repositories or git submodules, I want `gradusctl guard` to respect `.git` repository boundaries during workspace resolution, so that an independent non-Gradus repository inside a Gradus directory never inherits the parent's governance rules.

#### Acceptance Criteria

1. `R2.AC1` WHEN a PreToolUse payload originates inside a directory containing `.git` and no `gradus.yaml` whose parent directory contains `gradus.yaml`, the system SHALL treat the nested repository as outside Gradus governance and allow the action with exit code `0`.
   - Acceptance check: evaluating a `git commit` payload from a nested standalone `.git` repository inside a Gradus workspace returns `allow` (`{}` for Claude Code native payloads) with exit code `0` instead of inheriting the parent's rules.
2. `R2.AC2` WHEN a PreToolUse payload originates inside a regular subdirectory (without `.git`) of a Gradus repository, such as `.agents/` or `src/pkg`, the system SHALL resolve the enclosing Gradus repository root and enforce its rules.
   - Acceptance check: evaluating a protected-path or hook-rule payload from `.agents/` or a nested source subdirectory inside a Gradus repository resolves the enclosing `gradus.yaml` and returns the expected `ask`/`deny` decision.

### R3 Non-Blocking Hook Status on Malformed `gradus.yaml`

**User Story:** As a developer using SessionStart hooks, I want `gradusctl status --hook` and `gradusctl status --antigravity-hook` to stay non-blocking even when `gradus.yaml` has a syntax error, while keeping `gradusctl guard` fail-closed, so that broken configuration never breaks session startup and never weakens runtime enforcement.

#### Acceptance Criteria

1. `R3.AC1` IF `gradus.yaml` contains invalid YAML or fails schema parsing during `gradusctl status --hook` or `gradusctl status --antigravity-hook`, THEN the system SHALL emit a diagnostic summary indicating invalid configuration and exit with code `0` (D-82).
   - Acceptance check: invoking `gradusctl status --hook` and `gradusctl status --antigravity-hook` in a repository with syntactically broken `gradus.yaml` returns exit code `0` and outputs a diagnostic status string (`lock=invalid`) rather than exiting `2`.
2. `R3.AC2` IF `gradus.yaml` contains invalid YAML or fails schema parsing during non-hook `gradusctl status` or `gradusctl guard`, THEN the system SHALL exit with code `2`.
   - Acceptance check: non-hook `gradusctl status` and `gradusctl guard` return exit code `2` on malformed `gradus.yaml`.
3. `R3.AC3` The system SHALL document in `README.md` that a malformed `gradus.yaml` fails closed in `gradusctl guard` and requires manual repair.
   - Acceptance check: `README.md` explicitly notes manual repair for invalid `gradus.yaml` under Security boundaries.

### R4 MCP `propose_rule` Diagnostics and Design Closure (`O-1`)

**User Story:** As an operator and specification reader, I want accurate MCP validation error messages and a design document with no stale open items, so that diagnostics and documentation match actual product behavior.

#### Acceptance Criteria

1. `R4.AC1` WHEN `propose_rule` receives a `level` argument containing `..` versus a path separator (`/`, `\`) without `..`, the system SHALL reject both with distinct, accurate error messages (`path traversal is not allowed` for `..`, and `invalid level identifier` for path separators).
   - Acceptance check: `propose_rule` with `level: "../etc"` reports path traversal, whereas `level: "unknown/level"` reports an invalid level identifier without mislabeling it as path traversal.
2. `R4.AC2` The system SHALL close open item `O-1` and update section `24.3` in `docs/design/gradus-design.md` from `Proposed` to `Validated` referencing `wallet-tracker`, and record `v0.4.2` in `CHANGELOG.md`.
   - Acceptance check: `docs/design/gradus-design.md` marks §24.3 as validated on `wallet-tracker`, marks `O-1` closed in §27, and `CHANGELOG.md` contains a `## [0.4.2]` section.

## Non-Functional Requirements

- `NFR1` All changes MUST preserve `CGO_ENABLED=0` static compilation, zero new external Go dependencies (`go mod tidy -diff`), and race-free unit tests (`go test -race`).
- `NFR2` All repository artifacts MUST remain in canonical English (`PRJ-LANG-001`).

## Constraints And Dependencies

- `C1` `gradusctl guard` MUST remain fail-closed (exit `2`) on malformed `gradus.yaml` inside a Gradus repository (Constitution Hard Rule #4 and Threat Model T2: `gradus.yaml` is a protected governance file).
- `C2` No `git commit` or `git push` may occur without explicit user confirmation (`PRJ-GIT-001`).

## Out Of Scope

- Changing `serverInfo.version` format in `gradusctl serve --mcp` (retaining `v0.4.x` from `version.Get().Version`).
- Changing `gradusctl guard` behavior outside Gradus repositories (`D-89` intentionally returns `allow` when no `gradus.yaml` is present).
- Adding explicit profile-change banners beyond existing compiled-target drift detection.
