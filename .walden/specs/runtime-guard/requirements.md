---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T11:00:12Z
last_modified: 2026-11-20T11:00:12Z
approved_fingerprint: sha256:9e7207caba6ad3c1e4c76a31e10c78a6efe997c064b7c3eaa81cf68344808ee6
---

# Requirements Document

## Introduction

Feature **F5 (`runtime-guard`)** implements Gradus's deterministic runtime hook evaluator (`internal/guard` and `gradusctl guard`) to satisfy Phase 0 exit criterion **E3**, **ADR 0003** (Static Guard Dispatcher), and **Threat Model T2** (Anti-Tamper). When invoked by an AI coding agent's `PreToolUse` hook (Claude Code or Antigravity), `gradusctl guard` reads the tool invocation payload from `stdin`, evaluates all active `enforcement: hook` rules from the local resolved view (including the developer's `~/.gradus/levels/user` level) together with built-in anti-tamper protection, and outputs a structured decision (`allow`, `ask`, or `deny`) in under `100ms` without making any network calls.

## Requirements

### R1 Multi-Agent PreToolUse Payload Normalization

**User Story:** As an AI agent runtime integrator, I want `gradusctl guard` to parse and normalize `PreToolUse` JSON payloads from both Claude Code and Antigravity, so that a single static hook dispatcher works consistently across all supported agents.

#### Acceptance Criteria

1. `R1.AC1` WHEN `gradusctl guard` receives a valid Claude Code `PreToolUse` JSON payload on `stdin` (such as `Bash`, `Edit`, `Write`, or `MultiEdit` with `tool_input`), the system SHALL normalize the invocation into a canonical action class (`command` or `write`) and target string (shell command or normalized slash-separated path).
   - Acceptance check: Parsing Claude Code `Bash` and `Edit`/`Write` JSON payloads produces canonical `command` and `write` actions with extracted command text or repository-relative file path.
2. `R1.AC2` WHEN `gradusctl guard` receives a valid Antigravity `PreToolUse` JSON payload on `stdin` (such as `run_command`, `write_to_file`, `replace_file_content`, or `multi_replace_file_content` with `tool_input`), the system SHALL normalize the invocation into the same canonical action class (`command` or `write`) and target string.
   - Acceptance check: Parsing Antigravity `run_command` and file-editing JSON payloads produces identical canonical action classes and targets as equivalent Claude Code payloads.
3. `R1.AC3` WHEN `stdin` is empty, malformed JSON, or missing required tool identification, the system SHALL fail closed with exit code `2` and an actionable diagnostic error rather than returning `allow`.
   - Acceptance check: Feeding invalid JSON or empty input to `gradusctl guard` returns exit code `2` and does not emit an `allow` decision.

### R2 Built-In Anti-Tamper Protection (Threat Model T2)

**User Story:** As a repository maintainer, I want `gradusctl guard` to automatically guard Gradus governance and hook configuration files against silent agent modification, so that an agent cannot weaken or disable rules without explicit human confirmation.

#### Acceptance Criteria

1. `R2.AC1` WHEN a file-writing or file-editing tool targets any protected governance path (`gradus.yaml`, `gradus.lock`, `.gradus/**`, `.walden/**`, `.agents/hooks.json`, or `.claude/settings.json`), the system SHALL require at least `ask` (explicit human confirmation) even when no explicit user or project hook rule matches the path.
   - Acceptance check: Evaluating a `write` action targeting `gradus.yaml`, `gradus.lock`, `.gradus/levels/project/rules/R1.md`, `.walden/constitution.md`, `.agents/hooks.json`, or `.claude/settings.json` produces an `ask` decision with `anti_tamper: true` when no stricter `deny` rule applies.
2. `R2.AC2` WHEN both built-in anti-tamper (`ask`) and an explicit `enforcement: hook` rule with `decision: deny` match the same governance target, the system SHALL resolve the decision to `deny`.
   - Acceptance check: A `deny` hook rule targeting a governance file overrides the built-in `ask` floor and outputs `deny`.

### R3 Local Resolved View Guard Evaluation and Precedence

**User Story:** As a developer and team lead, I want `gradusctl guard` to evaluate all active project, overlay, and personal `user` hook rules locally with deterministic precedence (`deny > ask > allow`), so that personal guards work without dirtying git-tracked hook files (ADR 0003).

#### Acceptance Criteria

1. `R3.AC1` WHEN evaluating a normalized tool invocation against the local resolved view, the system SHALL load the repository pyramid together with `~/.gradus/levels/user` (when present) using local files/cache only and zero network calls.
   - Acceptance check: A `user`-level `enforcement: hook` rule (such as `ask` on `git commit`) is enforced by `gradusctl guard` without appearing in `.agents/hooks.json` or `.claude/settings.json`.
2. `R3.AC2` WHEN evaluating active, non-waived `enforcement: hook` rules, the system SHALL match `guard.action` (`command`/`bash` against shell commands, `write`/`edit` against file paths), `guard.pattern` (regular expression for commands; doublestar glob or regex for paths), and rule `scope` (when a target file path is known).
   - Acceptance check: Matching `command` and `write` hook rules trigger their configured decisions, while non-matching, waived, or non-`hook` rules do not trigger.
3. `R3.AC3` WHEN multiple hook rules or built-in anti-tamper match the same invocation, the system SHALL combine decisions using strict precedence `deny > ask > allow` and report all matched rule IDs and reasons.
   - Acceptance check: Given both an `ask` rule and a `deny` rule matching the same command, the final decision is `deny` and both rule IDs are recorded in `matched_rules`.

### R4 CLI Guard Command, Output Envelope, and Exit Codes

**User Story:** As a user and CI/agent harness operator, I want `gradusctl guard` to emit machine-readable hook decisions and adhere to Gradus's exit code contract (`0`/`1`/`2`), so that both agent runtimes and automated tests can consume its verdict reliably.

#### Acceptance Criteria

1. `R4.AC1` WHEN `gradusctl guard` completes evaluation, the system SHALL output a JSON payload containing `decision` (`allow` | `ask` | `deny`), `reason`, `matched_rules`, `anti_tamper`, and a `hookSpecificOutput` block (`hookEventName: "PreToolUse"`, `permissionDecision`, `permissionDecisionReason`).
   - Acceptance check: Running `gradusctl guard` with a valid payload prints valid JSON containing both top-level Gradus decision fields and `hookSpecificOutput`.
2. `R4.AC2` WHEN `gradusctl guard` exits, the system SHALL return exit code `0` for `allow` or `ask`, exit code `1` for `deny`, and exit code `2` for technical errors (malformed payload, invalid regex pattern, unreadable/invalid configuration, or lock tamper).
   - Acceptance check: End-to-end CLI tests verify exit `0` on `allow`, exit `0` on `ask` (including anti-tamper), exit `1` on `deny`, and exit `2` on invalid JSON or corrupted configuration.

## Non-Functional Requirements

- `NFR1` **Latency Budget:** Evaluating a typical local pyramid with `gradusctl guard` must complete in `< 100ms` (excluding cold OS process startup in CI containers) with zero network I/O.
- `NFR2` **Cross-Platform Path Safety:** Protected governance path matching must normalize Windows backslashes (`\`) and absolute paths rooted inside the workspace into clean relative slash paths so anti-tamper cannot be bypassed via `.\gradus.yaml`, `./gradus.yaml`, or absolute workspace paths.
- `NFR3` **Fail-Closed Technical Safety:** Configuration errors, lock tampering, or malformed hook payloads must return exit code `2` and never silently pass as `allow`.

## Constraints And Dependencies

- `C1` Must use only approved dependencies (`github.com/spf13/cobra`, `github.com/bmatcuk/doublestar/v4`, and Go standard library) and build with `CGO_ENABLED=0`.
- `C2` Must reuse `internal/resolve` (with `IncludeUser: true` and `OfflineOnly: true`) and `internal/schema` rather than duplicating pyramid or rule parsing logic.
- `C3` Must preserve ADR 0003: `.agents/hooks.json` and `.claude/settings.json` remain static dispatchers invoking `gradusctl guard`.

## Out Of Scope

- Writing evidence records on every `PreToolUse` hook invocation (kept out of the `< 100ms` hot path).
- Interactive terminal UI prompts inside `gradusctl guard` (human confirmation is rendered by the host agent runtime upon receiving `ask`).
- `PostToolUse` or `SessionStart` hook implementations (`SessionStart` already dispatches to `gradusctl status --brief` from F2/F3).
