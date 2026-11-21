---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-21T17:02:13Z
last_modified: 2026-11-21T17:02:13Z
approved_fingerprint: sha256:8acc43f9c9b49df9e244047cfa070141d351e282fa3388ce3be7d8688a0f105d
---

# Requirements Document

## Introduction

`doctor-and-guard-hardening` (`F11`) implements `gradusctl doctor` (§16) to diagnose configuration, `PATH`, and agent hook integrity (catching malformed `.claude/settings.json` / `.agents/hooks.json` entries), adds configurable `anti_tamper.mode` (`ask` | `deny`) in `gradus.yaml` for hard-blocking governance file mutations even in Claude Code `auto` mode, and adds hygiene commands (`gradusctl evidence prune`, `gradusctl lessons prune`) plus explicit user-level rule visibility in `gradusctl status`.

## Requirements

### R1 `gradusctl doctor` Diagnostics

**User Story:** As a developer setting up or debugging Gradus in a repository, I want `gradusctl doctor [--json]` to diagnose configuration, `PATH`, lockfile, and agent hook files with actionable fix hints, so that broken hook schemas or missing binaries are caught immediately.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl doctor [--json]` runs on a healthy repository with valid `gradus.yaml`, `gradus.lock`, `.claude/settings.json`, and `.agents/hooks.json`, the command SHALL report all checks passing and exit with code `0`.
   - Acceptance check: Running `gradusctl doctor --json` on a freshly compiled repository returns `"verdict": "pass"` and exit code `0`.
2. `R1.AC2` WHEN `.claude/settings.json` contains malformed JSON or an invalid hook `type` (not in `command | prompt | agent | http | mcp_tool`) or is missing the `gradusctl guard` hook entry, `gradusctl doctor` SHALL report a failing diagnostic with a fix hint and exit with code `1`.
   - Acceptance check: A `.claude/settings.json` containing `"type": "commbnd"` fails `gradusctl doctor --json` with exit code `1` and cites the invalid hook type and remediation.
3. `R1.AC3` WHEN `.agents/hooks.json` contains malformed JSON or is missing the `gradusctl guard` PreToolUse hook entry, `gradusctl doctor` SHALL report a failing diagnostic with a fix hint and exit with code `1`.
   - Acceptance check: Corrupting `.agents/hooks.json` fails `gradusctl doctor --json` with exit code `1` and cites `.agents/hooks.json`.

### R2 Configurable Anti-Tamper Policy (`ask` vs `deny`)

**User Story:** As a team lead whose developers run Claude Code in `auto` permission mode, I want to configure `anti_tamper.mode: deny` in `gradus.yaml`, so that agent edits or shell mutations targeting governance and hook files are hard-blocked rather than delegated to an auto-mode classifier.

#### Acceptance Criteria

1. `R2.AC1` WHEN `gradus.yaml` omits `anti_tamper` or sets `anti_tamper.mode: ask`, `gradusctl guard` SHALL preserve the default `ask` anti-tamper decision on protected governance paths and shell mutations.
   - Acceptance check: Without `anti_tamper`, `gradusctl guard` returns `ask` (`anti_tamper: true`) for `Edit` on `gradus.yaml`.
2. `R2.AC2` WHEN `gradus.yaml` sets `anti_tamper.mode: deny`, `gradusctl guard` SHALL return `deny` (`anti_tamper: true`) for writes or mutating shell commands targeting protected governance paths, emitting `permissionDecision: "deny"` (exit `0`) for native Claude Code payloads and `decision: "deny"` (exit `0`) for native Antigravity payloads.
   - Acceptance check: With `anti_tamper.mode: deny`, a native Claude Code `PreToolUse` payload editing `gradus.yaml` outputs `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny",...}}` with exit code `0`.
3. `R2.AC3` WHEN `gradus.yaml` sets an invalid `anti_tamper.mode` (anything other than `ask` or `deny`), schema validation SHALL fail closed with exit code `2`.
   - Acceptance check: `anti_tamper.mode: allow` is rejected by `ParsePyramidConfig` with a validation error.

### R3 Evidence/Lessons Hygiene and User-Level Rule Visibility

**User Story:** As a developer who runs trial checks and sessions, I want CLI commands to prune local evidence and lessons records and see active `user`-level guard rules in `gradusctl status`, so that state stays clean and user-level rules are transparent.

#### Acceptance Criteria

1. `R3.AC1` WHEN `gradusctl evidence prune [--before <YYYY-MM-DD> | --all]` is invoked, the CLI SHALL remove matching local JSON records under `.gradus/evidence/` and report the removed count.
   - Acceptance check: Running `gradusctl evidence prune --all --json` deletes local `.gradus/evidence/*.json` files and returns `"removed": N`.
2. `R3.AC2` WHEN `gradusctl lessons prune [--dismissed | --all]` is invoked, the CLI SHALL remove matching lesson JSON files from the lessons store and report the removed count.
   - Acceptance check: Running `gradusctl lessons prune --dismissed --json` deletes only dismissed lessons and preserves open lessons.
3. `R3.AC3` WHEN `gradusctl status --json` runs in a repository with personal `~/.gradus/levels/user` rules present, the JSON output SHALL include a `user_rules` summary count and IDs without increasing `status --brief` latency beyond `100ms`.
   - Acceptance check: `gradusctl status --json --user-dir <dir>` reports the loaded user-level rule IDs alongside lock/drift/conflict status.

## Non-Functional Requirements

- `NFR1` **Backward Compatibility:** `anti_tamper` is optional in `gradus.yaml` and defaults to `ask`; all existing Phase 0 and F9–F10 tests MUST continue to pass unchanged.
- `NFR2` **Performance & Static Build:** `gradusctl guard` and `gradusctl status --brief` MUST remain `< 100ms` offline; all code MUST be canonical English (`PRJ-LANG-001`) and build with `CGO_ENABLED=0` (`PRJ-GO-001`).

## Constraints And Dependencies

- `C1` `spec/v1/schema/gradus.schema.json` and `internal/schema/parse_config.go` MUST accept `anti_tamper.mode` (`ask | deny`) without emitting unknown-key notes.

## Out Of Scope

- Automatic self-healing edits inside `gradusctl doctor` (`doctor` diagnoses and explains fixes; `gradusctl compile` / `reconcile` applies changes).
