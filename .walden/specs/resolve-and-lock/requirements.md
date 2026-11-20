---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-19T22:39:18Z
last_modified: 2026-11-19T22:39:18Z
approved_fingerprint: sha256:041ac152455b6c43b29532f471d1604c730988bd510d5c6ca744989e7cae19c1
---

# Requirements Document

## Introduction

Feature **F2 (`resolve-and-lock`)** implements deterministic source fetching (local and `git`), normalized cross-platform lockfile hashing (`gradus.lock`), the §10 pyramid resolution algorithm (including inherited constraints, overlays, exceptions, the P2 restriction invariant, and structural conflict detection), and the `gradusctl resolve` and `gradusctl status` CLI commands.

## Requirements

### R1: Local and Git Source Resolution with Offline Policy

**User Story:** As a developer or CI runner, I want `gradusctl` to resolve both local levels and versioned Git levels using my existing Git credentials, so that shared rules are fetched reliably and offline failures are handled according to policy.

#### Acceptance Criteria

1. `R1.AC1` WHEN a level or overlay specifies `source.path`, the system SHALL resolve its directory on disk while skipping `~/` home-directory levels when the `CI` environment variable is set.
   - Acceptance check: local levels resolve to their workspace paths, and `~/` levels are omitted with a note when `CI=true`.
2. `R1.AC2` WHEN a level or overlay specifies `source.git` and `source.ref`, the system SHALL fetch the specified ref into `.gradus/cache/` using the system `git` binary via `os/exec` and record the resolved commit SHA.
   - Acceptance check: resolving a local bare or file-based Git repository clones/fetches the pinned tag/commit into `.gradus/cache/` and records its commit hash.
3. `R1.AC3` IF a remote `source.git` is unreachable and `offline.mode` is `cache` with an unexpired cached copy within `max_age`, THEN the system SHALL use the cached copy and emit a diagnostic note.
   - Acceptance check: when the remote URL is unreachable and a fresh cache entry exists within `max_age`, resolution succeeds with a cache-fallback note.
4. `R1.AC4` IF a remote `source.git` is unreachable and `offline.mode` is `fail-closed` OR the cache is missing/expired, THEN the system SHALL fail resolution and exit with code `2`.
   - Acceptance check: an unreachable `fail-closed` source or expired cache returns an error mapped to exit code `2`.

### R2: Normalized Hashing and `gradus.lock` Integrity

**User Story:** As a platform or security engineer, I want `gradus.lock` to pin resolved versions and normalized SHA-256 content hashes, so that tampering or unexpected drift in locked levels is detected deterministically across operating systems.

#### Acceptance Criteria

1. `R2.AC1` The system SHALL compute a deterministic SHA-256 hash over a level or overlay directory by sorting normalized slash paths lexicographically and normalizing `\r\n` line endings to `\n`.
   - Acceptance check: two directories with identical content differing only in CRLF vs LF line endings produce the exact same `sha256:` hash.
2. `R2.AC2` WHEN `gradusctl resolve --write` succeeds, the system SHALL write `gradus.lock` containing `schema: gradus/v1`, `definition_hash`, `entries` for all non-`user` levels and overlays, and `view_hash`.
   - Acceptance check: `gradus.lock` is written on disk, excludes the `user` level, and contains valid `sha256:` digests.
3. `R2.AC3` IF an existing `gradus.lock` is present and a `locked: true` level or overlay on disk or in cache has a hash differing from its recorded lock entry, THEN the system SHALL reject resolution as a lock integrity error and exit with code `2`.
   - Acceptance check: modifying a rule file inside a `locked: true` level without updating `gradus.lock` causes resolution to fail with exit code `2`.

### R3: Pyramid Resolution, Inherited Constraints, and Restriction Invariant

**User Story:** As a rule owner, I want `gradusctl` to merge levels and overlays in order, enforce inherited constraints, and block any rule that weakens a higher-level rule without permission, so that governance guarantees hold automatically.

#### Acceptance Criteria

1. `R3.AC1` WHEN resolving the pyramid, the system SHALL order levels as declared in `gradus.yaml`, insert each active overlay immediately after the level matching its `strength`, drop `retired` rules, exclude `draft` rules by default, and mark rules covered by unexpired exceptions in `.gradus/exceptions/` as waived for their scopes.
   - Acceptance check: active overlays appear right after their `strength` level in the resolved rule list, `retired`/`draft` rules are omitted, and active exceptions waive matching rules while expired exceptions emit warnings.
2. `R3.AC2` IF a lower-level definition violates inherited `constraints` from an upper level's `level.yaml` (`require_levels`, `max_extra_levels`, `allowed_overlays`), THEN the system SHALL report the constraint violation as a blocking conflict.
   - Acceptance check: omitting a required level or exceeding `max_extra_levels` produces a blocking conflict in the resolved view.
3. `R3.AC3` IF a lower-level rule declares `overrides: <ID>` targeting an unknown rule ID OR a committed level targets a `user`-level rule (ADR 0001), THEN the system SHALL fail resolution with exit code `2`.
   - Acceptance check: an `overrides` pointing to a nonexistent ID or a `project` rule overriding a `user` rule returns an error mapped to exit code `2`.
4. `R3.AC4` IF a rule duplicates an existing rule ID without `overrides`, OR weakens `severity`, widens `scope`, or weakens `enforcement` on a non-overridable rule, OR overrides a rule in a `locked: true` level not listed in `editable_rules`, THEN the system SHALL record a blocking conflict in the resolved view.
   - Acceptance check: each invalid specialization or duplicate ID appears in `conflicts` and causes `gradusctl resolve` to return exit code `1`.
5. `R3.AC5` WHEN confirmed semantic conflicts exist in `.gradus/findings/`, the system SHALL include them as blocking entries in `conflicts`.
   - Acceptance check: a confirmed finding in `.gradus/findings/` appears in `resolved_view.conflicts` and produces exit code `1`.

### R4: CLI Commands `gradusctl resolve` and `gradusctl status`

**User Story:** As a developer or agent hook, I want `gradusctl resolve` to produce the canonical resolved view and `gradusctl status --brief` to summarize repository state in under 100ms, so that compilers and session hooks operate reliably.

#### Acceptance Criteria

1. `R4.AC1` WHEN `gradusctl resolve` runs on a conflict-free pyramid, the system SHALL emit the resolved view (including `view_hash`, `lock_hash`, `levels`, `overlays`, `rules`, `conflicts`, `exceptions`, and `budget`) as text or `--json` and exit with code `0`.
   - Acceptance check: `gradusctl resolve --json` outputs valid JSON matching `spec/v1/schema/resolved-view.schema.json` and exits `0`.
2. `R4.AC2` WHEN `gradusctl resolve` detects one or more blocking conflicts, the system SHALL still emit the resolved view with `conflicts` populated and exit with code `1`.
   - Acceptance check: `gradusctl resolve --json` on a conflicting pyramid outputs the non-empty `conflicts` array and exits `1`.
3. `R4.AC3` WHEN `gradusctl status` or `gradusctl status --brief` runs in a Gradus repository, the system SHALL report lock state, conflict count, and pending lesson count in text or `--json` without making network calls.
   - Acceptance check: `gradusctl status --brief --json` completes in `< 100ms` on a local pyramid and exits `0` (or `1` when conflicts/drift are present).

## Non-Functional Requirements

- `NFR1` **Performance:** `gradusctl status --brief` MUST complete in under 100ms on local cached pyramids with zero network I/O.
- `NFR2` **Deterministic Subprocess Git (D-69):** Git source fetching MUST invoke the system `git` binary via `os/exec` with no CGO or `go-git` dependencies.
- `NFR3` **Exit Code Semantics (D-73):** `0` = clean pass, `1` = conflicts/violations, `2` = error (lock tamper, missing ref/unreachable `fail-closed` source, unknown override target, invalid schema).

## Constraints And Dependencies

- `C1` Builds on `internal/schema` from F1 (`core-schema-init`) and adds `github.com/bmatcuk/doublestar/v4` for scope glob comparison/matching.
- `C2` Implements ADR 0001 (CI override restriction) and ADR 0004 (container-derived level).

## Out Of Scope

- Generating target agent files (`AGENTS.md`, `CLAUDE.md`, `.agents/rules/`) (covered in **F3: `agent-compilers`**).
- Executing `gradusctl verify`, `check`, `evidence`, `hooks`, or `guard` (covered in **F4** and **F5**).
- Managing `.gradus/findings/` or `~/.gradus/lessons/` mutations (covered in **F6: `cli-lifecycle-state`**; F2 only reads existing `.gradus/findings/` and lesson counts).
