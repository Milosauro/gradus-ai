---
walden_schema_version: v1alpha1
status: approved
approved_at: 2026-11-20T09:28:07Z
last_modified: 2026-11-20T09:28:07Z
approved_fingerprint: sha256:d50c71049f40a7a8059bf4c0793319f121edf2c5e5564b5217c8be7fbb181021
---

# Requirements Document

## Introduction

Feature **F3 (`agent-compilers`)** implements deterministic, idempotent target compilers (`internal/compile`) and `gradusctl compile` for **Claude Code**, **Antigravity**, and **`AGENTS.md`**, including delimited managed sections, Antigravity `< 12,000`-character rule splitting, static hook dispatchers (ADR 0003), and embedded Agent Skills installation (`skills/embed.go`).

## Requirements

### R1: Managed Sections and `AGENTS.md` Compilation

**User Story:** As a developer, I want `gradusctl compile` to write generated instructions inside delimited managed sections, so that my own hand-written documentation in `AGENTS.md` and `CLAUDE.md` is never overwritten.

#### Acceptance Criteria

1. `R1.AC1` WHEN `internal/compile` updates a shared Markdown file (`AGENTS.md` or `CLAUDE.md`), the system SHALL insert or replace content strictly between `<!-- gradus:begin view_hash=... -->` and `<!-- gradus:end -->` markers while preserving all surrounding human-written content.
   - Acceptance check: running compilation on a file with custom prose above and below the markers updates only the delimited block and leaves custom prose byte-identical.
2. `R1.AC2` WHEN `agents-md` is configured in `targets`, the system SHALL render all active project-wide rules (excluding `user`-level rules) in pyramid order with rule ID, severity, provenance, text, and rejected rationalizations inside `AGENTS.md` (ADR 0002).
   - Acceptance check: `AGENTS.md` contains every active non-`user` rule regardless of `enforcement` tier and excludes `user`-level rules.

### R2: Claude Code Target Compilation (`claude-code`)

**User Story:** As a Claude Code user, I want `gradusctl compile` to generate `CLAUDE.md`, scoped `.claude/rules/`, `CLAUDE.local.md`, `.claude/settings.json`, and skills, so that Claude Code enforces my pyramid automatically.

#### Acceptance Criteria

1. `R2.AC1` WHEN `claude-code` is configured in `targets`, the system SHALL generate a `CLAUDE.md` managed section, scoped `.claude/rules/gradus-*.md` files with `paths` frontmatter for non-global rules, and `CLAUDE.local.md` for `user`-level rules.
   - Acceptance check: project rules appear in `CLAUDE.md` / `.claude/rules/gradus-*.md`, and `user`-level rules appear exclusively in `CLAUDE.local.md`.
2. `R2.AC2` WHEN `claude-code` is configured in `targets`, the system SHALL generate or merge `.claude/settings.json` with static `PreToolUse` (`gradusctl guard`) and `SessionStart` (`gradusctl status --brief`) hook dispatchers (ADR 0003) plus `permissions.deny`/`permissions.ask` entries for project `permission` rules without removing unrelated user settings.
   - Acceptance check: `.claude/settings.json` retains existing unrelated keys while registering the static Gradus hooks and permission rules.
3. `R2.AC3` WHEN `claude-code` is configured in `targets`, the system SHALL extract embedded Gradus skills from `skills/embed.go` into `.claude/skills/gradus-*/SKILL.md`.
   - Acceptance check: all embedded skill directories are written under `.claude/skills/` with valid `SKILL.md` contents.

### R3: Antigravity Target Compilation (`antigravity`)

**User Story:** As an Antigravity user, I want `gradusctl compile` to generate `.agents/rules/`, `.agents/hooks.json`, and skills within Antigravity's 12,000-character rule limit, so that rules load reliably without hitting truncation limits.

#### Acceptance Criteria

1. `R3.AC1` WHEN `antigravity` is configured in `targets`, the system SHALL generate an `AGENTS.md` managed section, project rule files under `.agents/rules/gradus-*.md`, and user-level rule files under `.agents/rules/gradus-user-*.md` with trigger frontmatter (`trigger: always_on` for global `**/*` rules, `trigger: glob` with `globs` for scoped rules).
   - Acceptance check: global and scoped rules are separated by trigger type, and `user`-level rules are written only to `gradus-user-*.md`.
2. `R3.AC2` The system SHALL split generated `.agents/rules/gradus-*.md` files whenever appending the next rule would exceed `11,500` characters, ensuring every generated file stays strictly below `12,000` characters (E1).
   - Acceptance check: compiling a rule set totaling over `25,000` characters produces multiple numbered `.agents/rules/gradus-*.md` files, each strictly `< 12,000` characters.
3. `R3.AC3` WHEN `antigravity` is configured in `targets`, the system SHALL generate or merge `.agents/hooks.json` with static `PreToolUse` (`gradusctl guard`) and `SessionStart` (`gradusctl status --brief`) dispatchers (ADR 0003) and extract embedded Gradus skills into `.agents/skills/gradus-*/SKILL.md`.
   - Acceptance check: `.agents/hooks.json` contains only static Gradus dispatchers (no user-level rule patterns) and `.agents/skills/` contains the embedded `SKILL.md` files.

### R4: `gradusctl compile` CLI, Idempotency, and Conflict Safety

**User Story:** As a developer or CI check, I want `gradusctl compile` to be strictly idempotent and refuse compilation when blocking conflicts exist, so that repeated runs produce zero git diffs.

#### Acceptance Criteria

1. `R4.AC1` WHEN `gradusctl compile` runs on a conflict-free pyramid, the system SHALL write all target outputs deterministically such that a second consecutive run produces zero byte differences on disk (E1), and exit with code `0`.
   - Acceptance check: running `gradusctl compile` twice on the same directory results in identical file contents and timestamps/hashes across both runs and exit code `0`.
2. `R4.AC2` IF the resolved pyramid contains one or more blocking conflicts, THEN `gradusctl compile` SHALL leave existing target files untouched and exit with code `1`.
   - Acceptance check: running `gradusctl compile` on a pyramid with a duplicate-ID conflict exits `1` without modifying `AGENTS.md` or `.agents/rules/`.
3. `R4.AC3` IF configuration, lockfile verification, or source resolution fails, THEN `gradusctl compile` SHALL report the error to stderr and exit with code `2`.
   - Acceptance check: running `gradusctl compile` with an invalid `gradus.yaml` or tampered locked level exits `2`.

## Non-Functional Requirements

- `NFR1` **Idempotency & Determinism (E1):** Output ordering (levels, rules, JSON keys) MUST be deterministic so consecutive `compile` runs produce zero diff across OSes.
- `NFR2` **Zero External Packaging Dependencies:** Skills are embedded via `go:embed` (`skills/embed.go`) with zero Node/npm runtime requirement.
- `NFR3` **Exit Code Semantics (D-73):** `0` = compiled cleanly, `1` = blocking conflicts prevented compilation, `2` = configuration/lock/I/O error.

## Constraints And Dependencies

- `C1` Builds on `internal/resolve` (`F2`) and `skills/embed.go` (`Wave 0B`).
- `C2` Implements ADR 0002 (all active rules in compiled instructions) and ADR 0003 (static hook dispatchers).

## Out Of Scope

- Checking drift on disk (`gradusctl verify`), running code checks (`gradusctl check`), and installing git hook managers (`gradusctl hooks`) (covered in **F4: `verify-check-evidence`**).
- Runtime evaluation inside `gradusctl guard` (covered in **F5: `runtime-guard`**).
- Authoring the final 6 MVP skills via skill-TDD (covered in **F7: `mvp-skills`**; F3 installs whatever skills are present in `skills/embed.go`).
