---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-25T16:18:16Z
last_modified: 2026-11-25T16:18:16Z
approved_fingerprint: sha256:d25899dbdcee201beafdab963cf3dc40cd18fcfdfb692969b5534799b8888d61
---

# Requirements Document

## Introduction

Feature `F29` (`enterprise-hardening-and-design-v1-2`, release `v0.4.1`) resolves critical enterprise deployment and robustness findings from the `v0.4.0` external audit and real-world Claude Code / Antigravity trials: making `gradusctl guard` and `gradusctl status --hook` safe in subdirectories and non-Gradus repositories when enterprise `managed-settings.json` is deployed machine-wide, hardening the MCP stdio server against oversized messages and `propose_rule` path traversal, fixing `serverInfo.version`, fixing historical `gradusctl trailers verify`, preventing `resolve --write` from overwriting `gradus.lock` on blocking conflicts, reducing anti-tamper false positives, and publishing **Design 1.2**.

## Requirements

### R1 Subdirectory & Non-Gradus Safety for `guard` and `status --hook`

**User Story:** As an IT administrator deploying `managed-settings.json` machine-wide, I want `gradusctl guard` and `gradusctl status --hook` to locate `gradus.yaml` from subdirectories and silently allow/succeed (exit `0`) in directories without Gradus so non-Gradus projects are never blocked.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl guard` runs inside a non-Gradus repository or non-repository directory, it SHALL exit `0` with an `allow` decision (empty JSON `{}` for Claude Code native payloads) rather than exiting `2`.
   - Acceptance check: `TestGuardAndStatusOutsideGradusV041` verifies exit `0` in a non-Gradus repo and in a non-git directory.
2. `R1.AC2` WHEN `gradusctl guard` runs inside a subdirectory of a Gradus repository, it SHALL walk up parent directories to locate `gradus.yaml` and evaluate that repository's rules and anti-tamper policy.
   - Acceptance check: `TestGuardAndStatusOutsideGradusV041` verifies a guard rule triggers when invoked from `sub/nested/`.
3. `R1.AC3` WHEN `gradusctl status --hook` or `gradusctl status --antigravity-hook` runs outside a Gradus repository, it SHALL exit `0` without error (D-82).
   - Acceptance check: `TestGuardAndStatusOutsideGradusV041` verifies exit `0` for `status --brief --hook` in a directory without `gradus.yaml`.

### R2 MCP Server Robustness, Path Containment, and Version Resolution

**User Story:** As an MCP client user, I want `gradusctl serve --mcp` to survive oversized messages without crashing, validate `propose_rule` levels against the pyramid, report the real resolved version, and expose read-only tool annotations.

#### Acceptance Criteria

1. `R2.AC1` WHEN an MCP client sends a JSON-RPC line exceeding the 4 MiB limit (e.g. 5 MB in `check_text`), `mcp.Server` SHALL return a JSON-RPC `-32600` error and continue serving subsequent requests on the same stdio session.
   - Acceptance check: `TestMCPHardeningV041` sends a >4 MiB line followed by `ping` and verifies both the `-32600` error and a successful `ping` response.
2. `R2.AC2` WHEN `propose_rule` receives a `level` that does not exist in the resolved pyramid or contains path traversal characters (e.g. `../../etc`), `mcp.Server` SHALL return `isError: true` without suggesting a path outside `.gradus/`.
   - Acceptance check: `TestMCPHardeningV041` verifies `../../etc` and `nonexistent` both return `isError: true`.
3. `R2.AC3` WHEN `initialize` or `tools/list` is called on `mcp.Server`, the server SHALL report `version.Get().Version` in `serverInfo.version`, echo any client-requested `protocolVersion`, and include read-only `annotations` (`readOnlyHint: true`, `destructiveHint: false`) on all 5 tools.
   - Acceptance check: `TestMCPHardeningV041` verifies negotiated `protocolVersion`, `version.Get().Version`, and read-only annotations on all 5 tools.

### R3 Historical Trailer Verification, `resolve --write` Guard, and Anti-Tamper Precision

**User Story:** As a maintainer, I want `trailers verify` to validate `Gradus-Lock-Hash` against the commit's own `gradus.lock`, `resolve --write` to refuse updating `gradus.lock` when blocking conflicts exist, and anti-tamper to avoid false positives on harmless `echo` or commit messages.

#### Acceptance Criteria

1. `R3.AC1` WHEN `trailers.VerifyCommit` inspects a commit carrying a `Gradus-Lock-Hash` trailer, it SHALL compare the trailer against `gradus.lock` at that commit (`git show <ref>:gradus.lock`, falling back to working tree) even without `--require-lock-hash`, rejecting forged hashes while passing valid historical commits after later lockfile updates.
   - Acceptance check: `TestTrailersHistoricalAndResolveGuardV041` verifies forged all-zero hash fails without `--require-lock-hash` and a valid older commit passes after `gradus.lock` changes.
2. `R3.AC2` WHEN `gradusctl resolve --write` detects blocking conflicts, it SHALL exit `1` without writing or overwriting `gradus.lock`.
   - Acceptance check: `TestTrailersHistoricalAndResolveGuardV041` verifies `gradus.lock` is untouched when `resolve --write` exits `1` on a conflict.
3. `R3.AC3` WHEN `guard.AntiTamperShellReason` evaluates shell commands, it SHALL allow harmless `echo SKIP=foo` and `git commit -m "fix HUSKY=0"` while blocking env-prefix bypasses (`HUSKY=0 git commit`, `LEFTHOOK_EXCLUDE=lint git commit`, `GIT_CONFIG_COUNT=1 git commit`) and writes to `.config/lefthook.yml`.
   - Acceptance check: `TestAntiTamperPrecisionV041` verifies both true positives and false-positive reductions.

### R4 Design 1.2 and Phase 3 Documentation Sync

**User Story:** As an auditor or adopter, I want `docs/design/gradus-design.md` (Design 1.2), `spec/v1/SPEC.md`, `docs/compatibility-matrix.md`, `README.md`, and `CHANGELOG.md` synchronized with all Phase 3 capabilities and decisions.

#### Acceptance Criteria

1. `R4.AC1` WHEN reading `docs/design/gradus-design.md`, `spec/v1/SPEC.md`, `docs/compatibility-matrix.md`, `README.md`, and `CHANGELOG.md`, those documents SHALL reflect Design 1.2 (§20/§21 active, decisions `D-84`–`D-89`), active `models` in `SPEC.md`, Phase 3 targets in `compatibility-matrix.md`, and `v0.4.1` plus `--model-tier` self-declaration in `README.md`/`CHANGELOG.md`.
   - Acceptance check: `TestDesign12SyncV041` verifies all five documentation files.

## Non-Functional Requirements

1. `NFR1` `gradusctl` SHALL compile as a static `CGO_ENABLED=0` binary invoking `git` via `os/exec` (`PRJ-GO-001`).
2. `NFR2` All repository files SHALL be written in canonical English (`PRJ-LANG-001`).

## Constraints

1. `C1` No breaking changes to existing `gradus/v1` schemas or CLI exit codes (`0`/`1`/`2`).
2. `C2` All existing unit and integration tests SHALL pass without regressions.
